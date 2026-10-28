"""
Stack único del reto: dos Lambdas en contenedor (Go y Node), cada una detrás de su API Gateway,
los secretos del JWT y la API key en Secrets Manager, y la app de Amplify para el frontend.

    Amplify (React) ──JWT──► API Gateway ─► Lambda Go (Fiber)  ──HTTP + JWT──► API Gateway ─► Lambda Node (Express)
                                              │ QR + validaciones                                 │ estadísticas
                                              └──────────── Secrets Manager (secreto JWT, API key) ┘
"""
from pathlib import Path

from aws_cdk import (
    CfnOutput,
    Duration,
    RemovalPolicy,
    Stack,
    aws_amplify as amplify,
    aws_apigateway as apigw,
    aws_ecr_assets as ecr_assets,
    aws_lambda as lambda_,
    aws_logs as logs,
    aws_secretsmanager as secretsmanager,
)
from constructs import Construct

RAIZ = Path(__file__).resolve().parents[2]  # carpeta reto-interseguro/
PREFIJO = "reto-matrices"


class RetoMatricesStack(Stack):

    def __init__(self, scope: Construct, construct_id: str, version: str, **kwargs) -> None:
        super().__init__(scope, construct_id, **kwargs)

        # ── Secretos ─────────────────────────────────────────────────────────
        # Se generan en el despliegue y nadie los escribe a mano. Las Lambdas los leen
        # una vez por contenedor; no quedan en texto plano en la configuración.
        secreto_jwt = secretsmanager.Secret(
            self, "JwtSecret",
            secret_name=f"{PREFIJO}/jwt-secret",
            description="Secreto HS256 compartido por las dos APIs para firmar y verificar el JWT",
            generate_secret_string=secretsmanager.SecretStringGenerator(password_length=64, exclude_punctuation=True),
            removal_policy=RemovalPolicy.DESTROY,
        )
        api_key = secretsmanager.Secret(
            self, "ApiKey",
            secret_name=f"{PREFIJO}/api-key",
            description="API key con la que el frontend pide el JWT (no hay login)",
            generate_secret_string=secretsmanager.SecretStringGenerator(password_length=32, exclude_punctuation=True),
            removal_policy=RemovalPolicy.DESTROY,
        )

        # ── API en Node (estadísticas) ───────────────────────────────────────
        fn_node = self._funcion(
            "ApiNodeFn", f"{PREFIJO}-api-node", "api-node",
            memoria=256, timeout=10,
            entorno={"JWT_SECRET_ARN": secreto_jwt.secret_arn, "VERSION": version},
        )
        secreto_jwt.grant_read(fn_node)
        api_node = self._api("ApiNode", f"{PREFIJO}-api-node", fn_node)

        # ── API en Go (QR) ───────────────────────────────────────────────────
        # Su timeout (20 s) queda por debajo del límite de API Gateway (29 s),
        # y el de su cliente hacia Node (8 s) por debajo de este.
        fn_go = self._funcion(
            "ApiGoFn", f"{PREFIJO}-api-go", "api-go",
            memoria=512, timeout=20,
            entorno={
                "NODE_API_URL": api_node.url,
                "JWT_SECRET_ARN": secreto_jwt.secret_arn,
                "API_KEY_ARN": api_key.secret_arn,
                "VERSION": version,
            },
        )
        secreto_jwt.grant_read(fn_go)
        api_key.grant_read(fn_go)
        api_go = self._api("ApiGo", f"{PREFIJO}-api-go", fn_go)

        # ── Frontend en Amplify (despliegue manual por zip, como DiseniaStudio) ──
        app_frontend = amplify.CfnApp(self, "Frontend", name=PREFIJO, platform="WEB")
        rama = amplify.CfnBranch(self, "FrontendMain", app_id=app_frontend.attr_app_id, branch_name="main", stage="PRODUCTION")

        CfnOutput(self, "ApiGoUrl", value=api_go.url)
        CfnOutput(self, "ApiNodeUrl", value=api_node.url)
        CfnOutput(self, "ApiKeySecretArn", value=api_key.secret_arn)
        CfnOutput(self, "AmplifyAppId", value=app_frontend.attr_app_id)
        CfnOutput(self, "AmplifyBranch", value=rama.branch_name)
        CfnOutput(self, "FrontendUrl", value=f"https://main.{app_frontend.attr_default_domain}")

    def _funcion(self, id_: str, nombre: str, carpeta: str, memoria: int, timeout: int, entorno: dict) -> lambda_.DockerImageFunction:
        """Lambda desde el Dockerfile del servicio (etapa `lambda`), con logs de una semana."""
        logs_fn = logs.LogGroup(
            self, f"{id_}Logs",
            log_group_name=f"/aws/lambda/{nombre}",
            retention=logs.RetentionDays.ONE_WEEK,
            removal_policy=RemovalPolicy.DESTROY,
        )
        return lambda_.DockerImageFunction(
            self, id_,
            function_name=nombre,
            code=lambda_.DockerImageCode.from_image_asset(
                str(RAIZ / carpeta), target="lambda", platform=ecr_assets.Platform.LINUX_AMD64,
            ),
            architecture=lambda_.Architecture.X86_64,
            memory_size=memoria,
            timeout=Duration.seconds(timeout),
            environment=entorno,
            log_group=logs_fn,
            tracing=lambda_.Tracing.ACTIVE,
        )

    def _api(self, id_: str, nombre: str, fn: lambda_.IFunction) -> apigw.LambdaRestApi:
        """API Gateway con un proxy hacia la Lambda: el enrutamiento lo hace Fiber o Express."""
        return apigw.LambdaRestApi(
            self, id_,
            handler=fn,
            rest_api_name=nombre,
            proxy=True,
            endpoint_types=[apigw.EndpointType.REGIONAL],
            deploy_options=apigw.StageOptions(
                stage_name="v1",
                throttling_rate_limit=20,   # no hay login: el límite de peticiones frena el abuso
                throttling_burst_limit=40,
                tracing_enabled=True,
            ),
            default_cors_preflight_options=apigw.CorsOptions(
                allow_origins=apigw.Cors.ALL_ORIGINS,  # el JWT va en un encabezado, no en cookies
                allow_methods=["GET", "POST", "OPTIONS"],
                allow_headers=["Content-Type", "Authorization", "X-Api-Key", "X-Request-ID"],
            ),
            # El rol de CloudWatch de API Gateway es uno por cuenta; este stack no lo toca.
            cloud_watch_role=False,
        )
