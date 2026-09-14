package auth

import (
	"testing"

	"github.com/google/uuid"
)

func TestJWTService(t *testing.T) {
	jwtService := NewJWTService("test-secret-key")
	userID := uuid.New()
	email := "test@example.com"
	role := "student"

	token, err := jwtService.GenerateToken(userID, email, role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Fatal("Expected non-empty token string")
	}

	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("Expected Email %s, got %s", email, claims.Email)
	}
	if claims.Role != role {
		t.Errorf("Expected Role %s, got %s", role, claims.Role)
	}
}

func TestPasswordHashing(t *testing.T) {
	password := "Password123!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !CheckPasswordHash(password, hash) {
		t.Error("Expected password hash check to succeed")
	}

	if CheckPasswordHash("WrongPassword", hash) {
		t.Error("Expected wrong password check to fail")
	}
}
