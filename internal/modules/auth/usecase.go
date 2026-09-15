// internal/modules/auth/usecase.go
package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/pkg/hash"
	"nexura-backend/pkg/jwt"
)

type AuthUsecase interface {
	Register(ctx context.Context, dto RegisterDTO) (*AuthResponse, error)
	Login(ctx context.Context, dto LoginDTO) (*AuthResponse, error)
	Logout(ctx context.Context) error
	GetProfile(ctx context.Context, userID uuid.UUID) (*User, error)
}

type authUsecase struct {
	userRepo   UserRepository
	jwtService *jwt.JWTService
}

func NewAuthUsecase(userRepo UserRepository, jwtService *jwt.JWTService) AuthUsecase {
	return &authUsecase{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

func (u *authUsecase) Register(ctx context.Context, dto RegisterDTO) (*AuthResponse, error) {
	existingUser, err := u.userRepo.GetByEmail(ctx, dto.Email)
	if err == nil && existingUser != nil {
		return nil, appErrors.ErrUserAlreadyExists
	}

	hashedPassword, err := hash.HashPassword(dto.Password)
	if err != nil {
		return nil, err
	}

	role := dto.Role
	if role == "" {
		role = RoleStudent
	}

	user := &User{
		FirstName:    dto.FirstName,
		LastName:     dto.LastName,
		Email:        dto.Email,
		PasswordHash: hashedPassword,
		Role:         role,
		Status:       StatusActive,
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	token, err := u.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (u *authUsecase) Login(ctx context.Context, dto LoginDTO) (*AuthResponse, error) {
	user, err := u.userRepo.GetByEmail(ctx, dto.Email)
	if err != nil {
		if errors.Is(err, appErrors.ErrUserNotFound) {
			return nil, appErrors.ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Status == StatusSuspended {
		return nil, appErrors.ErrUserSuspended
	}

	if !hash.CheckPasswordHash(dto.Password, user.PasswordHash) {
		return nil, appErrors.ErrInvalidCredentials
	}

	token, err := u.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: token,
		User:  user,
	}, nil
}

func (u *authUsecase) Logout(ctx context.Context) error {
	return nil
}

func (u *authUsecase) GetProfile(ctx context.Context, userID uuid.UUID) (*User, error) {
	return u.userRepo.GetByID(ctx, userID)
}
