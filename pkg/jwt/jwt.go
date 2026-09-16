package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaims struct {
	UserID uuid.UUID `json:"userId"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type JWTService struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewJWTService(secret string) *JWTService {
	return NewJWTServiceWithRefresh(secret, secret+"-refresh", 7*24*time.Hour, 30*24*time.Hour)
}

func NewJWTServiceWithRefresh(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *JWTService {
	if accessTTL <= 0 {
		accessTTL = 7 * 24 * time.Hour
	}
	if refreshTTL <= 0 {
		refreshTTL = 30 * 24 * time.Hour
	}
	return &JWTService{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

func (s *JWTService) AccessTTL() time.Duration  { return s.accessTTL }
func (s *JWTService) RefreshTTL() time.Duration { return s.refreshTTL }

func (s *JWTService) GenerateToken(userID uuid.UUID, email, role string) (string, error) {
	return s.sign(s.accessSecret, userID, email, role, s.accessTTL)
}

func (s *JWTService) GenerateRefreshToken(userID uuid.UUID, email, role string) (string, error) {
	return s.sign(s.refreshSecret, userID, email, role, s.refreshTTL)
}

func (s *JWTService) sign(secret []byte, userID uuid.UUID, email, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	return s.parse(tokenString, s.accessSecret)
}

func (s *JWTService) ValidateRefreshToken(tokenString string) (*JWTClaims, error) {
	return s.parse(tokenString, s.refreshSecret)
}

func (s *JWTService) parse(tokenString string, secret []byte) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.UserID == uuid.Nil && claims.Subject != "" {
		if id, err := uuid.Parse(claims.Subject); err == nil {
			claims.UserID = id
		}
	}
	return claims, nil
}
