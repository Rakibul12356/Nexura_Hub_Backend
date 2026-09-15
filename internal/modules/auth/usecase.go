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
	UpdateProfile(ctx context.Context, userID uuid.UUID, dto UpdateProfileDTO) (*User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, dto ChangePasswordDTO) error
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, dto ResetPasswordDTO) error
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

func (u *authUsecase) UpdateProfile(ctx context.Context, userID uuid.UUID, dto UpdateProfileDTO) (*User, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if dto.FirstName != nil {
		user.FirstName = *dto.FirstName
	}
	if dto.LastName != nil {
		user.LastName = *dto.LastName
	}
	if dto.Bio != nil {
		user.Bio = dto.Bio
	}
	if dto.Occupation != nil {
		user.Occupation = dto.Occupation
	}
	if dto.Phone != nil {
		user.Phone = dto.Phone
	}
	if dto.Website != nil {
		user.Website = dto.Website
	}
	if dto.Avatar != nil {
		user.Avatar = dto.Avatar
	}
	if err := u.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (u *authUsecase) ChangePassword(ctx context.Context, userID uuid.UUID, dto ChangePasswordDTO) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !hash.CheckPasswordHash(dto.CurrentPassword, user.PasswordHash) {
		return appErrors.ErrInvalidCredentials
	}
	hashedPassword, err := hash.HashPassword(dto.NewPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hashedPassword
	return u.userRepo.Update(ctx, user)
}

func (u *authUsecase) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	claims, err := u.jwtService.ValidateToken(refreshToken)
	if err != nil {
		return nil, err
	}
	user, err := u.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
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

func (u *authUsecase) ForgotPassword(ctx context.Context, email string) error {
	_, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return appErrors.ErrUserNotFound
	}
	return nil
}

func (u *authUsecase) ResetPassword(ctx context.Context, dto ResetPasswordDTO) error {
	return nil
}
