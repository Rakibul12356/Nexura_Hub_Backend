package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	DBUrl             string
	JWTSecret         string
	JWTRefreshSecret  string
	JWTAccessExpires  string
	JWTRefreshExpires string
	Environment       string
	ClientOrigin      string
	DummyPayments     bool
}

func LoadConfig() *Config {
	_ = godotenv.Load(".env")

	port := firstNonEmpty(os.Getenv("PORT"), "5000")
	dbUrl := firstNonEmpty(os.Getenv("DATABASE_URL"), os.Getenv("DB_URL"), "postgres://postgres:postgres@localhost:5432/nexura_hub?sslmode=disable")
	jwtSecret := firstNonEmpty(os.Getenv("JWT_ACCESS_SECRET"), os.Getenv("JWT_SECRET"), "nexura-super-secret-jwt-key-2026")
	jwtRefresh := firstNonEmpty(os.Getenv("JWT_REFRESH_SECRET"), jwtSecret+"-refresh")
	env := firstNonEmpty(os.Getenv("ENV"), os.Getenv("ENVIRONMENT"), "development")
	origin := firstNonEmpty(os.Getenv("CLIENT_ORIGIN"), "*")
	dummy := os.Getenv("DUMMY_PAYMENTS")
	dummyOn := dummy == "" || dummy == "true" || dummy == "1"

	return &Config{
		Port:              port,
		DBUrl:             dbUrl,
		JWTSecret:         jwtSecret,
		JWTRefreshSecret:  jwtRefresh,
		JWTAccessExpires:  firstNonEmpty(os.Getenv("JWT_ACCESS_EXPIRES"), "7d"),
		JWTRefreshExpires: firstNonEmpty(os.Getenv("JWT_REFRESH_EXPIRES"), "30d"),
		Environment:       env,
		ClientOrigin:      origin,
		DummyPayments:     dummyOn,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
