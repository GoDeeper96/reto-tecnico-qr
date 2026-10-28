package estadisticas

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"reto-interseguro/api-go/internal/matriz"
)

func TestReenviaTokenYReintentaUnaVezAnte5xx(t *testing.T) {
	intentos := 0
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		intentos++
		if r.Header.Get("Authorization") != "Bearer abc" || r.Header.Get("X-Request-ID") != "req-1" {
			t.Errorf("encabezados no reenviados: %v", r.Header)
		}
		if intentos == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var cuerpo struct {
			Matrices map[string]matriz.Matriz `json:"matrices"`
		}
		_ = json.NewDecoder(r.Body).Decode(&cuerpo)
		_ = json.NewEncoder(w).Encode(Resultado{CantidadValores: len(cuerpo.Matrices["q"])})
	}))
	defer servidor.Close()

	cliente := NuevoClienteHTTP(servidor.URL, time.Second)
	cliente.espera = time.Millisecond
	resultado, err := cliente.Calcular(context.Background(), map[string]matriz.Matriz{"q": {{1}, {2}}},
		Credenciales{Authorization: "Bearer abc", RequestID: "req-1"})
	if err != nil {
		t.Fatal(err)
	}
	if intentos != 2 || resultado.CantidadValores != 2 {
		t.Errorf("intentos %d, resultado %+v", intentos, resultado)
	}
}

func TestNoReintentaAnte4xx(t *testing.T) {
	intentos := 0
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		intentos++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer servidor.Close()

	_, err := NuevoClienteHTTP(servidor.URL, time.Second).Calcular(context.Background(), nil, Credenciales{})
	var remoto *ErrorRemoto
	if !errors.As(err, &remoto) || remoto.Estado != http.StatusUnauthorized || intentos != 1 {
		t.Fatalf("err %v, intentos %d", err, intentos)
	}
}
