package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"reto-interseguro/api-go/internal/auth"
	"reto-interseguro/api-go/internal/estadisticas"
	"reto-interseguro/api-go/internal/matriz"
)

// clienteFalso reemplaza a la API en Node y recuerda lo que recibió.
type clienteFalso struct {
	llamadas int
	cred     estadisticas.Credenciales
	falla    bool
}

func (f *clienteFalso) Calcular(_ context.Context, m map[string]matriz.Matriz, cred estadisticas.Credenciales) (estadisticas.Resultado, error) {
	f.llamadas++
	f.cred = cred
	if f.falla {
		return estadisticas.Resultado{}, errors.New("conexión rechazada")
	}
	return estadisticas.Resultado{CantidadValores: len(m["q"]) * len(m["q"][0])}, nil
}

func preparar(t *testing.T, falla bool) (*Servidor, *clienteFalso, func(*http.Request) (*http.Response, map[string]any)) {
	t.Helper()
	cliente := &clienteFalso{falla: falla}
	tokens := auth.NuevoTokens("secreto-de-prueba", "clave-de-prueba")
	app := NuevaApp(tokens, cliente, "test")
	enviar := func(req *http.Request) (*http.Response, map[string]any) {
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		datos, _ := io.ReadAll(resp.Body)
		var cuerpo map[string]any
		_ = json.Unmarshal(datos, &cuerpo)
		return resp, cuerpo
	}
	return &Servidor{tokens: tokens}, cliente, enviar
}

func token(t *testing.T, s *Servidor) string {
	t.Helper()
	tk, _, err := s.tokens.Emitir("test")
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + tk
}

func peticion(cuerpo, autorizacion string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qr", strings.NewReader(cuerpo))
	req.Header.Set("Content-Type", "application/json")
	if autorizacion != "" {
		req.Header.Set("Authorization", autorizacion)
	}
	return req
}

func TestQRCorrectaLlamaALaAPIDeEstadisticas(t *testing.T) {
	s, cliente, enviar := preparar(t, false)
	auth := token(t, s)
	resp, cuerpo := enviar(peticion(`{"matriz": [[1,2],[3,4],[5,6]]}`, auth))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("estado %d: %v", resp.StatusCode, cuerpo)
	}
	if cuerpo["metodo"] != "householder" {
		t.Errorf("método por defecto: %v", cuerpo["metodo"])
	}
	v := cuerpo["validaciones"].(map[string]any)
	for _, regla := range []string{"qOrtogonal", "rTriangular", "reconstruccion"} {
		if !v[regla].(map[string]any)["cumple"].(bool) {
			t.Errorf("la regla %s no se cumple", regla)
		}
	}
	if cliente.llamadas != 1 || cliente.cred.Authorization != auth || cliente.cred.RequestID == "" {
		t.Errorf("la API de estadísticas no recibió el token y el request id: %+v", cliente)
	}
}

func TestSinTokenResponde401(t *testing.T) {
	_, _, enviar := preparar(t, false)
	resp, cuerpo := enviar(peticion(`{"matriz": [[1]]}`, ""))
	if resp.StatusCode != http.StatusUnauthorized || resp.Header.Get("Content-Type") != "application/problem+json" {
		t.Fatalf("estado %d, tipo %q: %v", resp.StatusCode, resp.Header.Get("Content-Type"), cuerpo)
	}
	resp, _ = enviar(peticion(`{"matriz": [[1]]}`, "Bearer token.falso.x"))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("token falso: estado %d", resp.StatusCode)
	}
}

func TestEntradasInvalidasResponden400(t *testing.T) {
	s, cliente, enviar := preparar(t, false)
	auth := token(t, s)
	for _, cuerpo := range []string{
		`no es json`,
		`{"otra": 1}`,
		`{"matriz": []}`,
		`{"matriz": [[1,2],[3]]}`,
		`{"matriz": [[1,"a"]]}`,
		`{"matriz": [[1,null]]}`,
		`{"matriz": [[1,2]], "metodo": "lu"}`,
		`{"matriz": [[1,2,3],[4,5,6]], "metodo": "gram-schmidt-clasico"}`,
	} {
		resp, r := enviar(peticion(cuerpo, auth))
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: estado %d, se esperaba 400: %v", cuerpo, resp.StatusCode, r)
		}
	}
	if cliente.llamadas != 0 {
		t.Error("no se debe llamar a la API de estadísticas con una entrada inválida")
	}
}

func TestValidacionFallidaResponde422SinLlamarANode(t *testing.T) {
	s, cliente, enviar := preparar(t, false)
	hilbert := make([][]float64, 10)
	for i := range hilbert {
		hilbert[i] = make([]float64, 10)
		for j := range hilbert[i] {
			hilbert[i][j] = 1 / float64(i+j+1)
		}
	}
	datos, _ := json.Marshal(map[string]any{"matriz": hilbert, "metodo": "gram-schmidt-clasico"})
	resp, cuerpo := enviar(peticion(string(datos), token(t, s)))
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("estado %d: %v", resp.StatusCode, cuerpo)
	}
	if cliente.llamadas != 0 {
		t.Error("con validaciones fallidas no se debe llamar a la API de estadísticas")
	}
}

func TestFallaDeNodeResponde502(t *testing.T) {
	s, _, enviar := preparar(t, true)
	resp, _ := enviar(peticion(`{"matriz": [[1,2],[3,4]]}`, token(t, s)))
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("estado %d, se esperaba 502", resp.StatusCode)
	}
}

func TestEmitirToken(t *testing.T) {
	_, _, enviar := preparar(t, false)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", nil)
	req.Header.Set("X-Api-Key", "clave-incorrecta")
	if resp, _ := enviar(req); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("API key incorrecta: estado %d", resp.StatusCode)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/token", nil)
	req.Header.Set("X-Api-Key", "clave-de-prueba")
	resp, cuerpo := enviar(req)
	if resp.StatusCode != http.StatusOK || cuerpo["accessToken"] == "" {
		t.Fatalf("estado %d: %v", resp.StatusCode, cuerpo)
	}
	// El token emitido sirve para llamar a /qr.
	resp, _ = enviar(peticion(`{"matriz": [[1]]}`, "Bearer "+cuerpo["accessToken"].(string)))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("con el token emitido: estado %d", resp.StatusCode)
	}
}

func TestAutovalores(t *testing.T) {
	s, _, enviar := preparar(t, false)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/autovalores", strings.NewReader(`{"matriz": [[2,1],[1,2]]}`))
	req.Header.Set("Authorization", token(t, s))
	resp, cuerpo := enviar(req)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("estado %d: %v", resp.StatusCode, cuerpo)
	}
	valores := cuerpo["autovalores"].([]any)
	if len(valores) != 2 || valores[0].(float64) < 2.99 {
		t.Errorf("autovalores inesperados: %v", valores)
	}
}
