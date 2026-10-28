// Package auth emite y verifica los JWT que protegen las dos APIs.
//
// No hay login: el frontend obtiene un token de corta duración presentando la API key.
// Las dos APIs comparten el secreto de firma (HS256), y la API en Go reenvía el mismo token a la de Node.
package auth

import (
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Emisor    = "reto-matrices"
	Audiencia = "reto-matrices-apis"
	Duracion  = 15 * time.Minute
)

// Tokens emite y verifica tokens con un secreto compartido.
type Tokens struct {
	secreto []byte
	apiKey  []byte
	ahora   func() time.Time
}

func NuevoTokens(secreto, apiKey string) *Tokens {
	return &Tokens{secreto: []byte(secreto), apiKey: []byte(apiKey), ahora: time.Now}
}

// ApiKeyValida compara en tiempo constante para no filtrar la clave por la duración de la comparación.
func (t *Tokens) ApiKeyValida(clave string) bool {
	return len(t.apiKey) > 0 && subtle.ConstantTimeCompare([]byte(clave), t.apiKey) == 1
}

// Emitir crea un token firmado para el sujeto indicado.
func (t *Tokens) Emitir(sujeto string) (string, time.Duration, error) {
	ahora := t.ahora()
	claims := jwt.RegisteredClaims{
		Issuer:    Emisor,
		Audience:  jwt.ClaimStrings{Audiencia},
		Subject:   sujeto,
		IssuedAt:  jwt.NewNumericDate(ahora),
		ExpiresAt: jwt.NewNumericDate(ahora.Add(Duracion)),
	}
	firmado, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secreto)
	return firmado, Duracion, err
}

var ErrSinToken = errors.New("falta el encabezado Authorization: Bearer <token>")

// Verificar valida firma, algoritmo, emisor, audiencia y caducidad.
func (t *Tokens) Verificar(encabezado string) (*jwt.RegisteredClaims, error) {
	token, ok := strings.CutPrefix(encabezado, "Bearer ")
	if !ok || token == "" {
		return nil, ErrSinToken
	}
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) { return t.secreto, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Emisor),
		jwt.WithAudience(Audiencia),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.ahora),
	)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
