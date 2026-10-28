// Package estadisticas es el puerto hacia la API en Node y su adaptador HTTP.
package estadisticas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"reto-interseguro/api-go/internal/matriz"
)

// Resultado es la respuesta de POST /api/v1/estadisticas de la API en Node.
type Resultado struct {
	Maximo          float64         `json:"maximo"`
	Minimo          float64         `json:"minimo"`
	Suma            float64         `json:"suma"`
	Promedio        float64         `json:"promedio"`
	CantidadValores int             `json:"cantidadValores"`
	EsDiagonal      map[string]bool `json:"esDiagonal"`
	AlgunaDiagonal  bool            `json:"algunaDiagonal"`
}

// Credenciales que se reenvían a la API en Node: el mismo token del usuario y el id de la petición.
type Credenciales struct {
	Authorization string
	RequestID     string
}

// Cliente es el puerto que usa el handler; en las pruebas se reemplaza por uno falso.
type Cliente interface {
	Calcular(ctx context.Context, matrices map[string]matriz.Matriz, cred Credenciales) (Resultado, error)
}

// ClienteHTTP llama a la API en Node por HTTP.
type ClienteHTTP struct {
	urlBase    string
	http       *http.Client
	reintentos int
	espera     time.Duration
}

func NuevoClienteHTTP(urlBase string, timeout time.Duration) *ClienteHTTP {
	return &ClienteHTTP{urlBase: urlBase, http: &http.Client{Timeout: timeout}, reintentos: 1, espera: 200 * time.Millisecond}
}

// ErrorRemoto es una respuesta de error de la API en Node.
type ErrorRemoto struct {
	Estado int
	Cuerpo string
}

func (e *ErrorRemoto) Error() string {
	return fmt.Sprintf("la API de estadísticas respondió %d: %s", e.Estado, e.Cuerpo)
}

// Calcular envía las matrices. Reintenta una vez ante errores de red o 5xx:
// el cálculo no tiene efectos secundarios, así que repetirlo es seguro.
func (c *ClienteHTTP) Calcular(ctx context.Context, matrices map[string]matriz.Matriz, cred Credenciales) (Resultado, error) {
	cuerpo, err := json.Marshal(map[string]any{"matrices": matrices})
	if err != nil {
		return Resultado{}, err
	}
	var ultimo error
	for intento := 0; intento <= c.reintentos; intento++ {
		if intento > 0 {
			select {
			case <-ctx.Done():
				return Resultado{}, ctx.Err()
			case <-time.After(c.espera):
			}
		}
		resultado, reintentable, err := c.enviar(ctx, cuerpo, cred)
		if err == nil {
			return resultado, nil
		}
		ultimo = err
		if !reintentable {
			break
		}
	}
	return Resultado{}, ultimo
}

func (c *ClienteHTTP) enviar(ctx context.Context, cuerpo []byte, cred Credenciales) (Resultado, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.urlBase+"/api/v1/estadisticas", bytes.NewReader(cuerpo))
	if err != nil {
		return Resultado{}, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", cred.Authorization)
	if cred.RequestID != "" {
		req.Header.Set("X-Request-ID", cred.RequestID)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Resultado{}, true, err
	}
	defer resp.Body.Close()
	datos, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Resultado{}, true, err
	}
	if resp.StatusCode != http.StatusOK {
		return Resultado{}, resp.StatusCode >= 500, &ErrorRemoto{Estado: resp.StatusCode, Cuerpo: string(datos)}
	}
	var resultado Resultado
	if err := json.Unmarshal(datos, &resultado); err != nil {
		return Resultado{}, false, fmt.Errorf("respuesta inválida de la API de estadísticas: %w", err)
	}
	return resultado, false, nil
}
