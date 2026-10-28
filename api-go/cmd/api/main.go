// Punto de entrada de la API en Go. El mismo binario corre como servidor HTTP (local o Docker)
// o como función Lambda detrás de API Gateway, según el entorno.
package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	fiberadapter "github.com/awslabs/aws-lambda-go-api-proxy/fiber"

	"reto-interseguro/api-go/internal/auth"
	"reto-interseguro/api-go/internal/config"
	"reto-interseguro/api-go/internal/estadisticas"
	"reto-interseguro/api-go/internal/httpapi"
)

func main() {
	cfg, err := config.Cargar(context.Background())
	if err != nil {
		log.Fatalf("configuración: %v", err)
	}
	app := httpapi.NuevaApp(
		auth.NuevoTokens(cfg.SecretoJWT, cfg.APIKey),
		estadisticas.NuevoClienteHTTP(cfg.URLEstadisticas, cfg.TimeoutEstadistica),
		cfg.Version,
	)

	// Lambda define AWS_LAMBDA_RUNTIME_API; fuera de Lambda se levanta un servidor normal.
	if os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" {
		lambda.Start(fiberadapter.New(app).ProxyWithContext)
		return
	}
	log.Fatal(app.Listen(":" + cfg.Puerto))
}
