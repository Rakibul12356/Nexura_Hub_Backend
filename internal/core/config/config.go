// internal/core/config/config.go
package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DBUrl       string
	JWTSecret   string
	Environment string
}

func LoadConfig() *Config {
	_ = godotenv.Load(".env")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@localhost:5432/nexura_hub?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "nexura-super-secret-jwt-key-2026"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	return &Config{
		Port:        port,
		DBUrl:       dbUrl,
		JWTSecret:   jwtSecret,
		Environment: env,
	}
}
