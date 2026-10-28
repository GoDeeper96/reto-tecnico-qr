// Package config lee la configuración de variables de entorno (12-factor).
// En AWS los secretos vienen de Secrets Manager; en local, directamente de variables de entorno.
package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type Config struct {
	Puerto             string
	URLEstadisticas    string        // URL base de la API en Node
	TimeoutEstadistica time.Duration // menor que el timeout de la Lambda y de API Gateway (29 s)
	SecretoJWT         string
	APIKey             string
	Version            string
}

// Cargar arma la configuración. Si existen JWT_SECRET_ARN / API_KEY_ARN, lee los secretos de Secrets Manager.
func Cargar(ctx context.Context) (Config, error) {
	c := Config{
		Puerto:             valor("PORT", "8080"),
		URLEstadisticas:    strings.TrimRight(os.Getenv("NODE_API_URL"), "/"),
		TimeoutEstadistica: 8 * time.Second,
		SecretoJWT:         os.Getenv("JWT_SECRET"),
		APIKey:             os.Getenv("API_KEY"),
		Version:            valor("VERSION", "dev"),
	}
	if c.URLEstadisticas == "" {
		return c, fmt.Errorf("falta la variable NODE_API_URL")
	}
	arnJWT, arnAPIKey := os.Getenv("JWT_SECRET_ARN"), os.Getenv("API_KEY_ARN")
	if arnJWT != "" || arnAPIKey != "" {
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return c, fmt.Errorf("configuración de AWS: %w", err)
		}
		cliente := secretsmanager.NewFromConfig(awsCfg)
		if c.SecretoJWT, err = leerSecreto(ctx, cliente, arnJWT, c.SecretoJWT); err != nil {
			return c, err
		}
		if c.APIKey, err = leerSecreto(ctx, cliente, arnAPIKey, c.APIKey); err != nil {
			return c, err
		}
	}
	if c.SecretoJWT == "" || c.APIKey == "" {
		return c, fmt.Errorf("faltan JWT_SECRET y API_KEY (o JWT_SECRET_ARN y API_KEY_ARN)")
	}
	return c, nil
}

func leerSecreto(ctx context.Context, cliente *secretsmanager.Client, arn, porDefecto string) (string, error) {
	if arn == "" {
		return porDefecto, nil
	}
	salida, err := cliente.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(arn)})
	if err != nil {
		return "", fmt.Errorf("leer el secreto %s: %w", arn, err)
	}
	return aws.ToString(salida.SecretString), nil
}

func valor(nombre, porDefecto string) string {
	if v := os.Getenv(nombre); v != "" {
		return v
	}
	return porDefecto
}
