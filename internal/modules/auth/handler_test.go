package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockAuthUsecase struct{}

func (m *mockAuthUsecase) Register(ctx context.Context, dto RegisterDTO) (*AuthResponse, error) {
	return nil, nil
}

func (m *mockAuthUsecase) Login(ctx context.Context, dto LoginDTO) (*AuthResponse, error) {
	return nil, nil
}

func (m *mockAuthUsecase) Logout(ctx context.Context) error {
	return nil
}

func (m *mockAuthUsecase) GetProfile(ctx context.Context, userID uuid.UUID) (*User, error) {
	return nil, nil
}

func (m *mockAuthUsecase) UpdateProfile(ctx context.Context, userID uuid.UUID, dto UpdateProfileDTO) (*User, error) {
	return nil, nil
}

func (m *mockAuthUsecase) ChangePassword(ctx context.Context, userID uuid.UUID, dto ChangePasswordDTO) error {
	return nil
}

func (m *mockAuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	return nil, nil
}

func (m *mockAuthUsecase) ForgotPassword(ctx context.Context, email string) error {
	return nil
}

func (m *mockAuthUsecase) ResetPassword(ctx context.Context, dto ResetPasswordDTO) error {
	return nil
}

func TestLogoutHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuthHandler(&mockAuthUsecase{})

	router := gin.New()
	router.POST("/api/v1/auth/logout", handler.Logout)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/logout", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	expectedBody := `{"message":"Logged out successfully","status":"success","success":true}`
	if w.Body.String() != expectedBody {
		t.Errorf("expected body %s, got %s", expectedBody, w.Body.String())
	}
}
