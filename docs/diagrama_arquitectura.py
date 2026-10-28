"""Genera docs/arquitectura.drawio (editable en draw.io) con la arquitectura serverless del reto.

Uso:      python docs/diagrama_arquitectura.py
Exportar: draw.io.exe -x -f png -s 2 -b 16 -o docs/arquitectura.png docs/arquitectura.drawio
"""
import xml.etree.ElementTree as ET
from pathlib import Path

TINTA, GRIS = "#232F3E", "#5A6B7B"
NARANJA, ROSA, ROJO, MORADO = "#ED7100", "#E7157B", "#DD344C", "#C925D1"
FLECHA = "#3B4B5C"

raiz = ET.Element("mxfile", host="drawio")
diagrama = ET.SubElement(raiz, "diagram", name="Arquitectura serverless", id="arquitectura")
modelo = ET.SubElement(diagrama, "mxGraphModel", grid="1", gridSize="10", page="0", math="0", shadow="0")
celdas = ET.SubElement(modelo, "root")
ET.SubElement(celdas, "mxCell", id="0")
ET.SubElement(celdas, "mxCell", id="1", parent="0")
cont = [0]


def celda(estilo, x=0, y=0, w=0, h=0, valor="", vertice=True, origen=None, destino=None, puntos=()):
    cont[0] += 1
    cid = f"c{cont[0]}"
    attrs = {"id": cid, "value": valor, "style": estilo, "parent": "1"}
    if vertice:
        attrs["vertex"] = "1"
    else:
        attrs.update(edge="1", source=origen, target=destino)
    c = ET.SubElement(celdas, "mxCell", attrs)
    geo = ET.SubElement(c, "mxGeometry", **({"x": str(x), "y": str(y), "width": str(w), "height": str(h)} if vertice else {"relative": "1"}),
                        **{"as": "geometry"})
    if puntos:
        arreglo = ET.SubElement(geo, "Array", **{"as": "points"})
        for px, py in puntos:
            ET.SubElement(arreglo, "mxPoint", x=str(px), y=str(py))
    return cid


def texto(x, y, w, h, html, alin="center", tam=12, color=TINTA):
    return celda(f"text;html=1;whiteSpace=wrap;align={alin};verticalAlign=middle;fontFamily=Arial;fontSize={tam};"
                 f"fontColor={color};", x, y, w, h, html)


def grupo(x, y, w, h, etiqueta, icono, color, discontinuo=False):
    return celda(f"points=[];outlineConnect=0;gradientColor=none;html=1;whiteSpace=wrap;fontSize=13;fontStyle=1;"
                 f"fontFamily=Arial;container=0;pointerEvents=0;collapsible=0;recursiveResize=0;shape=mxgraph.aws4.group;"
                 f"grIcon={icono};strokeColor={color};fillColor=none;verticalAlign=top;align=left;spacingLeft=30;"
                 f"fontColor={color};dashed={1 if discontinuo else 0};", x, y, w, h, etiqueta)


def caja(x, y, w, h, etiqueta, color, alin="left"):
    """Rectángulo redondeado que agrupa los recursos de una API."""
    return celda(f"rounded=1;arcSize=4;html=1;whiteSpace=wrap;fillColor={color}0D;strokeColor={color};strokeWidth=1.5;"
                 f"dashed=1;dashPattern=6 4;verticalAlign=top;align={alin};spacingLeft=12;spacingRight=12;spacingTop=6;fontFamily=Arial;"
                 f"fontSize=14;fontStyle=1;fontColor={color};", x, y, w, h, etiqueta)


def icono_aws(x, y, recurso, color, etiqueta, lado="bottom", ancho_etiqueta=170, tam=64):
    posicion = ("verticalLabelPosition=bottom;verticalAlign=top;align=center;" if lado == "bottom"
                else "labelPosition=right;verticalLabelPosition=middle;verticalAlign=middle;align=left;spacingLeft=8;")
    estilo = (f"sketch=0;outlineConnect=0;fontColor={TINTA};gradientColor=none;fillColor={color};strokeColor=#ffffff;"
              f"dashed=0;{posicion}html=1;whiteSpace=wrap;fontSize=12;fontFamily=Arial;aspect=fixed;"
              f"shape=mxgraph.aws4.resourceIcon;resIcon=mxgraph.aws4.{recurso};")
    cid = celda(estilo, x, y, tam, tam, etiqueta)
    return cid


def flecha(origen, destino, etiqueta="", discontinua=False, puntos=(), salida=None, entrada=None, color=FLECHA):
    estilo = (f"edgeStyle=orthogonalEdgeStyle;rounded=1;html=1;endArrow=block;endFill=1;strokeColor={color};"
              f"strokeWidth={1.3 if discontinua else 1.8};fontFamily=Arial;fontSize=11.5;fontColor={TINTA};"
              f"labelBackgroundColor=#FFFFFF;{'dashed=1;dashPattern=5 4;' if discontinua else ''}")
    for nombre, punto in (("exit", salida), ("entry", entrada)):
        if punto:
            estilo += f"{nombre}X={punto[0]};{nombre}Y={punto[1]};{nombre}Dx=0;{nombre}Dy=0;"
    return celda(estilo, valor=etiqueta, vertice=False, origen=origen, destino=destino, puntos=puntos)


# ── Título ───────────────────────────────────────────────────────────────────
texto(20, 0, 1400, 34, "<b>Reto técnico — arquitectura serverless en AWS</b>", alin="left", tam=20)
texto(20, 30, 1400, 24, "Dos APIs en AWS Lambda (imágenes Docker) detrás de API Gateway · frontend en Amplify · "
      "infraestructura como código con AWS CDK", alin="left", tam=13, color=GRIS)

# ── Cliente ──────────────────────────────────────────────────────────────────
usuario = celda(f"sketch=0;outlineConnect=0;gradientColor=none;fillColor={TINTA};strokeColor=none;dashed=0;html=1;"
                f"aspect=fixed;shape=mxgraph.aws4.client;", 40, 350, 70, 60)
texto(15, 414, 120, 36, "<b>Navegador</b><br>React (SPA)")

# ── Nube de AWS y región ─────────────────────────────────────────────────────
grupo(260, 70, 1170, 720, "AWS Cloud · cuenta EYATECH", "mxgraph.aws4.group_aws_cloud_alt", TINTA)
grupo(285, 108, 1125, 662, "us-east-1 · stack RetoMatrices (AWS CDK)", "mxgraph.aws4.group_region", "#00A4A6", discontinuo=True)

amplify = icono_aws(330, 145, "amplify", ROJO, "")
texto(404, 145, 230, 64, "<b>Amplify Hosting</b><br>frontend React + Vite<br>rama main", alin="left")
cdk = icono_aws(1060, 145, "cloudformation", ROSA, "")
texto(1134, 145, 260, 64, "<b>AWS CDK → CloudFormation</b><br>infraestructura como código<br>deploy.ps1", alin="left")

# ── API en Go ────────────────────────────────────────────────────────────────
caja_go = caja(315, 290, 530, 210, "API en Go — QR + validaciones", NARANJA, alin="right")
gw_go = icono_aws(345, 350, "api_gateway", ROSA, "")
texto(320, 418, 114, 60, "<b>API Gateway</b><br>REST · etapa v1<br>20 req/s · CORS", tam=11.5)
fn_go = icono_aws(505, 350, "lambda", NARANJA, "")
texto(579, 330, 262, 104, "<b>Lambda · Go 1.26 + Fiber</b><br>imagen Docker (provided.al2023)<br>"
      "QR: Householder · Givens · Gram-Schmidt<br>valida: Q ortogonal · R triangular<br>y Q·R = A", alin="left", tam=11.5)

# ── API en Node ──────────────────────────────────────────────────────────────
caja_node = caja(880, 290, 520, 210, "API en Node — estadísticas", "#2E7D32", alin="right")
gw_node = icono_aws(910, 350, "api_gateway", ROSA, "")
texto(885, 418, 114, 60, "<b>API Gateway</b><br>REST · etapa v1<br>20 req/s · CORS", tam=11.5)
fn_node = icono_aws(1070, 350, "lambda", NARANJA, "")
texto(1144, 330, 250, 104, "<b>Lambda · Node 22 + Express 5</b><br>imagen Docker (nodejs22.x)<br>"
      "máximo · mínimo · suma · promedio<br>¿alguna matriz es diagonal?", alin="left", tam=11.5)

# ── Servicios compartidos por las dos Lambdas ────────────────────────────────
compartidos = caja(315, 600, 1085, 140, "Servicios que usan las dos Lambdas", GRIS)
icono_aws(350, 645, "ecr", NARANJA, "", tam=56)
texto(414, 645, 230, 56, "<b>ECR</b><br>imágenes Docker de las Lambdas", alin="left", tam=11.5)
icono_aws(690, 645, "secrets_manager", ROJO, "", tam=56)
texto(754, 640, 250, 66, "<b>Secrets Manager</b><br>secreto JWT (HS256) y API key,<br>generados por CDK, leídos al arrancar",
      alin="left", tam=11.5)
icono_aws(1050, 645, "cloudwatch_2", ROSA, "", tam=56)
texto(1114, 645, 280, 56, "<b>CloudWatch Logs · X-Ray</b><br>logs JSON con requestId · trazas", alin="left", tam=11.5)

# ── Flujo principal ──────────────────────────────────────────────────────────
flecha(usuario, amplify, "① carga la página", puntos=[(75, 177)], salida=(0.5, 0), entrada=(0, 0.5))
flecha(usuario, gw_go, "", salida=(1, 0.5), entrada=(0, 0.5))
texto(110, 300, 145, 66, "② POST /auth/token<br>(X-Api-Key)<br>③ POST /qr (Bearer JWT)", tam=11.5)
flecha(gw_go, fn_go, "proxy", salida=(1, 0.5), entrada=(0, 0.5))
flecha(fn_go, gw_node, "④ POST /estadisticas · mismo JWT + X-Request-ID", puntos=[(537, 255), (942, 255)],
       salida=(0.5, 0), entrada=(0.5, 0))
flecha(gw_node, fn_node, "proxy", salida=(1, 0.5), entrada=(0, 0.5))
flecha(usuario, caja_node, "GET /health — el frontend consulta las dos APIs", discontinua=True,
       puntos=[(150, 404), (150, 545), (920, 545)], salida=(1, 0.9), entrada=(0.077, 1), color=GRIS)

# ── Dependencias ─────────────────────────────────────────────────────────────
flecha(compartidos, caja_go, "", discontinua=True, salida=(0.3, 0), entrada=(0.62, 1), color=GRIS)
flecha(compartidos, caja_node, "", discontinua=True, salida=(0.75, 0), entrada=(0.6, 1), color=GRIS)

# ── Leyenda ──────────────────────────────────────────────────────────────────
texto(20, 800, 1410, 26,
      "<b>Leyenda:</b> línea continua = petición HTTP del flujo principal (① → ④) · línea discontinua = consulta de "
      "estado o dependencia de las Lambdas (imagen, secretos y logs)", alin="left", tam=11.5, color=GRIS)

salida = Path(__file__).with_name("arquitectura.drawio")
ET.ElementTree(raiz).write(salida, encoding="utf-8", xml_declaration=False)
print(salida)
