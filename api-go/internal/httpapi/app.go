// Package httpapi es el adaptador HTTP (Fiber) de la API en Go.
package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"reto-interseguro/api-go/internal/auth"
	"reto-interseguro/api-go/internal/estadisticas"
	"reto-interseguro/api-go/internal/matriz"
)

// Servidor agrupa las dependencias de los handlers (inyectadas por constructor).
type Servidor struct {
	tokens       *auth.Tokens
	estadisticas estadisticas.Cliente
	version      string
}

// NuevaApp arma la aplicación Fiber con sus middlewares y rutas.
func NuevaApp(tokens *auth.Tokens, est estadisticas.Cliente, version string) *fiber.App {
	s := &Servidor{tokens: tokens, estadisticas: est, version: version}
	app := fiber.New(fiber.Config{
		AppName:      "api-go",
		BodyLimit:    2 << 20, // 2 MB: una matriz de 200x200 en JSON ocupa menos de 1 MB
		ErrorHandler: manejarError,
	})
	app.Use(recover.New())
	app.Use(requestid.New()) // reutiliza el X-Request-ID recibido o crea uno nuevo
	app.Use(logger.New(logger.Config{
		Format: `{"requestId":"${respHeader:X-Request-ID}","metodo":"${method}","ruta":"${path}","estado":${status},"latencia":"${latency}"}` + "\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:  "*", // la autenticación va en un encabezado Bearer, no en cookies
		AllowHeaders:  "Content-Type, Authorization, X-Api-Key, X-Request-ID",
		ExposeHeaders: "X-Request-ID",
	}))

	app.Get("/health", s.salud)
	v1 := app.Group("/api/v1")
	v1.Post("/auth/token", s.emitirToken)
	v1.Post("/qr", s.requiereToken, s.factorizar)
	v1.Post("/autovalores", s.requiereToken, s.autovalores)
	return app
}

func (s *Servidor) salud(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"estado": "ok", "servicio": "api-go", "version": s.version})
}

// emitirToken entrega un JWT de corta duración a quien presente la API key.
func (s *Servidor) emitirToken(c *fiber.Ctx) error {
	if !s.tokens.ApiKeyValida(c.Get("X-Api-Key")) {
		return problema(fiber.StatusUnauthorized, "API key inválida", "Envíe la API key en el encabezado X-Api-Key")
	}
	token, duracion, err := s.tokens.Emitir("frontend")
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"accessToken": token, "tokenType": "Bearer", "expiresIn": int(duracion.Seconds())})
}

func (s *Servidor) requiereToken(c *fiber.Ctx) error {
	if _, err := s.tokens.Verificar(c.Get(fiber.HeaderAuthorization)); err != nil {
		return problema(fiber.StatusUnauthorized, "No autorizado", err.Error())
	}
	return c.Next()
}

type peticionMatriz struct {
	Matriz [][]*float64 `json:"matriz"` // punteros para distinguir null de 0
	Metodo string       `json:"metodo"`
}

// leerMatriz decodifica y valida el cuerpo. Cualquier problema es un 400.
func leerMatriz(c *fiber.Ctx) (matriz.Matriz, string, error) {
	var p peticionMatriz
	if err := json.Unmarshal(c.Body(), &p); err != nil || p.Matriz == nil {
		return nil, "", problema(fiber.StatusBadRequest, "Cuerpo inválido",
			"El cuerpo debe ser un JSON con el campo 'matriz': un array de arrays de números")
	}
	a := make(matriz.Matriz, len(p.Matriz))
	for i, fila := range p.Matriz {
		a[i] = make([]float64, len(fila))
		for j, valor := range fila {
			if valor == nil {
				return nil, "", problema(fiber.StatusBadRequest, "Matriz inválida", fmt.Sprintf("El elemento [%d][%d] no es un número", i, j))
			}
			a[i][j] = *valor
		}
	}
	if err := matriz.Validar(a); err != nil {
		return nil, "", aProblema(err)
	}
	return a, p.Metodo, nil
}

func aProblema(err error) error {
	var entrada *matriz.ErrorEntrada
	if errors.As(err, &entrada) {
		return problema(fiber.StatusBadRequest, "Matriz inválida", entrada.Mensaje)
	}
	return err
}

// factorizar calcula la QR, la valida y envía Q y R a la API de estadísticas.
func (s *Servidor) factorizar(c *fiber.Ctx) error {
	a, metodo, err := leerMatriz(c)
	if err != nil {
		return err
	}
	if metodo == "" {
		metodo = matriz.MetodoPorDefecto
	}
	inicio := time.Now()
	q, r, err := matriz.FactorizarQR(a, metodo)
	if err != nil {
		return aProblema(err)
	}
	validaciones := matriz.ValidarQR(a, q, r)
	duracionQR := time.Since(inicio)
	if !validaciones.TodasCumplen() {
		p := problema(fiber.StatusUnprocessableEntity, "La factorización no cumple las validaciones",
			fmt.Sprintf("Con %s, Q no es ortogonal, R no es triangular o Q·R no reconstruye A dentro de la tolerancia %.0e. "+
				"La matriz está mal condicionada para este método: use householder o givens.", metodo, matriz.Tolerancia))
		p.Extra = map[string]any{"metodo": metodo, "validaciones": validaciones}
		return p
	}

	inicio = time.Now()
	resultado, err := s.estadisticas.Calcular(c.UserContext(), map[string]matriz.Matriz{"q": q, "r": r}, estadisticas.Credenciales{
		Authorization: c.Get(fiber.HeaderAuthorization),
		RequestID:     c.GetRespHeader(fiber.HeaderXRequestID),
	})
	if err != nil {
		return problema(fiber.StatusBadGateway, "No se pudieron calcular las estadísticas", err.Error())
	}
	filas, columnas := a.Dimensiones()
	return c.JSON(fiber.Map{
		"metodo":       metodo,
		"dimensiones":  fiber.Map{"filas": filas, "columnas": columnas},
		"q":            q,
		"r":            r,
		"validaciones": validaciones,
		"estadisticas": resultado,
		"duracionMs": fiber.Map{
			"qr":           float64(duracionQR.Microseconds()) / 1000,
			"estadisticas": float64(time.Since(inicio).Microseconds()) / 1000,
		},
	})
}

// autovalores es un endpoint extra: autovalores y autovectores de una matriz simétrica con el algoritmo QR.
func (s *Servidor) autovalores(c *fiber.Ctx) error {
	a, _, err := leerMatriz(c)
	if err != nil {
		return err
	}
	resultado, err := matriz.CalcularAutovalores(a)
	if err != nil {
		return aProblema(err)
	}
	return c.JSON(resultado)
}
