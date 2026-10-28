package httpapi

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
)

// Problema es una respuesta de error con el formato RFC 7807 (Problem Details), igual en las dos APIs.
type Problema struct {
	Tipo      string         `json:"type"`
	Titulo    string         `json:"title"`
	Estado    int            `json:"status"`
	Detalle   string         `json:"detail,omitempty"`
	RequestID string         `json:"requestId,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
}

func (p *Problema) Error() string { return p.Titulo + ": " + p.Detalle }

func problema(estado int, titulo, detalle string) *Problema {
	return &Problema{Tipo: "about:blank", Titulo: titulo, Estado: estado, Detalle: detalle}
}

// manejarError convierte cualquier error en un Problema. Los errores no previstos se registran y responden 500
// sin exponer el detalle interno.
func manejarError(c *fiber.Ctx, err error) error {
	var p *Problema
	var errFiber *fiber.Error
	switch {
	case errors.As(err, &p):
	case errors.As(err, &errFiber):
		p = problema(errFiber.Code, errFiber.Message, "")
	default:
		log.Printf("error no controlado en %s %s: %v", c.Method(), c.Path(), err)
		p = problema(fiber.StatusInternalServerError, "Error interno", "Ocurrió un error inesperado")
	}
	p.RequestID = c.GetRespHeader(fiber.HeaderXRequestID)
	if p.Estado == fiber.StatusUnauthorized {
		c.Set(fiber.HeaderWWWAuthenticate, `Bearer realm="reto-matrices"`)
	}
	return c.Status(p.Estado).JSON(p, "application/problem+json")
}
