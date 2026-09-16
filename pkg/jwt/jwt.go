package jwt

import (
	"encoding/json"
	"errors"
	"strings"
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

func (c *JWTClaims) UnmarshalJSON(data []byte) error {
	var raw struct {
		UserID json.RawMessage `json:"userId"`
		Email  string          `json:"email"`
		Role   string          `json:"role"`
		jwt.RegisteredClaims
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Email = raw.Email
	c.Role = raw.Role
	c.RegisteredClaims = raw.RegisteredClaims
	c.UserID = parseUUIDValue(raw.UserID)
	c.normalize()
	return nil
}

func (c *JWTClaims) normalize() {
	if c.UserID == uuid.Nil && c.Subject != "" {
		if id, err := uuid.Parse(strings.TrimSpace(c.Subject)); err == nil {
			c.UserID = id
		}
	}
	c.Role = strings.ToLower(strings.TrimSpace(c.Role))
	c.Email = strings.TrimSpace(c.Email)
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
		Role:   strings.ToLower(strings.TrimSpace(role)),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Minute)),
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

func (s *JWTService) ValidateAccessOrRefresh(tokenString string) (*JWTClaims, error) {
	if claims, err := s.ValidateToken(tokenString); err == nil {
		return claims, nil
	}
	return s.ValidateRefreshToken(tokenString)
}

func (s *JWTService) parse(tokenString string, secret []byte) (*JWTClaims, error) {
	tokenString = stripToken(tokenString)
	if tokenString == "" {
		return nil, errors.New("empty token")
	}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithLeeway(2*time.Minute),
	)
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	}

	claims := &JWTClaims{}
	token, err := parser.ParseWithClaims(tokenString, claims, keyFunc)
	if err == nil && token.Valid {
		claims.normalize()
		if claims.UserID != uuid.Nil {
			return claims, nil
		}
	}

	mapTok, mapErr := parser.Parse(tokenString, keyFunc)
	if mapErr != nil {
		if err != nil {
			return nil, err
		}
		return nil, mapErr
	}
	mc, ok := mapTok.Claims.(jwt.MapClaims)
	if !ok || !mapTok.Valid {
		return nil, errors.New("invalid token")
	}
	out := claimsFromMap(mc)
	if out.UserID == uuid.Nil {
		return nil, errors.New("invalid token subject")
	}
	return out, nil
}

func stripToken(tokenString string) string {
	tokenString = strings.TrimSpace(tokenString)
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	tokenString = strings.TrimPrefix(tokenString, "bearer ")
	return strings.TrimSpace(tokenString)
}

func claimsFromMap(mc jwt.MapClaims) *JWTClaims {
	c := &JWTClaims{}
	c.Email, _ = mc["email"].(string)
	c.Role, _ = mc["role"].(string)
	if sub, _ := mc["sub"].(string); sub != "" {
		c.Subject = sub
	}
	c.UserID = parseUUIDAny(mc["userId"])
	if c.UserID == uuid.Nil {
		c.UserID = parseUUIDAny(mc["user_id"])
	}
	if c.UserID == uuid.Nil {
		c.UserID = parseUUIDAny(mc["id"])
	}
	c.normalize()
	return c
}

func parseUUIDValue(raw json.RawMessage) uuid.UUID {
	if len(raw) == 0 || string(raw) == "null" {
		return uuid.Nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return parseUUIDAny(s)
	}
	return uuid.Nil
}

func parseUUIDAny(v any) uuid.UUID {
	switch t := v.(type) {
	case uuid.UUID:
		return t
	case string:
		id, err := uuid.Parse(strings.TrimSpace(t))
		if err == nil {
			return id
		}
	}
	return uuid.Nil
}
