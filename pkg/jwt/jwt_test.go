package jwt

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestGenerateAndValidateRoundTrip(t *testing.T) {
	svc := NewJWTService("test-secret-key")
	userID := uuid.MustParse("d4a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	token, err := svc.GenerateToken(userID, "tapas@nexurahub.com", "Instructor")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("access token just issued must validate: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("userId=%s want %s", claims.UserID, userID)
	}
	if claims.Role != "instructor" {
		t.Fatalf("role=%q", claims.Role)
	}
}

func TestValidateSubjectWhenUserIdMissing(t *testing.T) {
	svc := NewJWTService("test-secret-key")
	userID := uuid.MustParse("f659784d-1f7f-44f5-9e32-c1f7bb5aafc5")
	now := time.Now()
	raw := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
		"sub":   userID.String(),
		"email": "admin@nexurahub.com",
		"role":  "admin",
		"exp":   now.Add(time.Hour).Unix(),
		"iat":   now.Unix(),
	})
	token, err := raw.SignedString([]byte("test-secret-key"))
	if err != nil {
		t.Fatal(err)
	}
	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("sub-only token must validate: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("userId from sub=%s want %s", claims.UserID, userID)
	}
}

func TestValidateAccessOrRefresh(t *testing.T) {
	svc := NewJWTService("test-secret-key")
	userID := uuid.New()
	refresh, err := svc.GenerateRefreshToken(userID, "a@b.com", "admin")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := svc.ValidateAccessOrRefresh(refresh)
	if err != nil {
		t.Fatalf("refresh must be accepted as session token: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("userId=%s", claims.UserID)
	}
}

func TestStripBearerPrefix(t *testing.T) {
	svc := NewJWTService("test-secret-key")
	userID := uuid.New()
	token, _ := svc.GenerateToken(userID, "a@b.com", "student")
	claims, err := svc.ValidateToken("Bearer " + token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != userID {
		t.Fatalf("userId=%s", claims.UserID)
	}
}
