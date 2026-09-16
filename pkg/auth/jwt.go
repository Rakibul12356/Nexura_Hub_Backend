package auth

import "nexura-backend/pkg/jwt"

type JWTClaims = jwt.JWTClaims
type JWTService = jwt.JWTService

func NewJWTService(secret string) *JWTService {
	return jwt.NewJWTService(secret)
}
