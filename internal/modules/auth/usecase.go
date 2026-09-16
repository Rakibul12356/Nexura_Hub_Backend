package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/pkg/hash"
	"nexura-backend/pkg/jwt"
	"nexura-backend/pkg/utils"
)

type AuthUsecase interface {
	Register(ctx context.Context, dto RegisterDTO) (*AuthResponse, error)
	Login(ctx context.Context, dto LoginDTO) (*AuthResponse, error)
	Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error
	GetProfile(ctx context.Context, userID uuid.UUID) (*User, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, dto UpdateProfileDTO) (*User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, dto ChangePasswordDTO) error
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
	ForgotPassword(ctx context.Context, email string) (string, error)
	ResetPassword(ctx context.Context, dto ResetPasswordDTO) error
	VerifyEmail(ctx context.Context, token string) error
	GetInstructorPublic(ctx context.Context, id uuid.UUID) (*InstructorPublic, error)
}

type authUsecase struct {
	userRepo   UserRepository
	jwtService *jwt.JWTService
}

func NewAuthUsecase(userRepo UserRepository, jwtService *jwt.JWTService) AuthUsecase {
	return &authUsecase{userRepo: userRepo, jwtService: jwtService}
}

func (u *authUsecase) issueTokens(ctx context.Context, user *User) (*AuthResponse, error) {
	token, err := u.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, err
	}
	refresh, err := u.jwtService.GenerateRefreshToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, err
	}
	_ = u.userRepo.SaveRefreshToken(ctx, user.ID, utils.HashToken(refresh), time.Now().Add(u.jwtService.RefreshTTL()))
	return &AuthResponse{User: user.Public(), Token: token, RefreshToken: refresh}, nil
}

func (u *authUsecase) Register(ctx context.Context, dto RegisterDTO) (*AuthResponse, error) {
	if dto.ConfirmPassword != "" && dto.ConfirmPassword != dto.Password {
		return nil, errors.New("passwords do not match")
	}
	role := dto.Role
	if role == "" {
		role = RoleStudent
	}
	if role == RoleAdmin {
		return nil, appErrors.ErrInvalidRole
	}
	if role != RoleStudent && role != RoleInstructor {
		return nil, errors.New("role must be student or instructor")
	}

	existing, err := u.userRepo.GetByEmail(ctx, strings.ToLower(dto.Email))
	if err == nil && existing != nil {
		return nil, appErrors.ErrUserAlreadyExists
	}

	hashed, err := hash.HashPassword(dto.Password)
	if err != nil {
		return nil, err
	}
	user := &User{
		FirstName:    dto.FirstName,
		LastName:     dto.LastName,
		Email:        strings.ToLower(dto.Email),
		PasswordHash: hashed,
		Role:         role,
		Status:       StatusActive,
	}
	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	if role == RoleInstructor {
		_ = u.userRepo.EnsureWallet(ctx, "instructor", user.ID)
	}
	verifyTok := utils.RandomToken()
	_ = u.userRepo.SaveEmailVerify(ctx, user.ID, utils.HashToken(verifyTok), time.Now().Add(48*time.Hour))
	return u.issueTokens(ctx, user)
}

func (u *authUsecase) Login(ctx context.Context, dto LoginDTO) (*AuthResponse, error) {
	user, err := u.userRepo.GetByEmail(ctx, strings.ToLower(dto.Email))
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
	return u.issueTokens(ctx, user)
}

func (u *authUsecase) Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	if refreshToken != "" {
		_ = u.userRepo.DeleteRefreshToken(ctx, utils.HashToken(refreshToken))
	}
	if userID != uuid.Nil {
		_ = u.userRepo.DeleteUserRefreshTokens(ctx, userID)
	}
	return nil
}

func (u *authUsecase) GetProfile(ctx context.Context, userID uuid.UUID) (*User, error) {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user.Public(), nil
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
	if dto.Email != nil && *dto.Email != "" {
		user.Email = strings.ToLower(*dto.Email)
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
	if dto.Designation != nil {
		user.Designation = dto.Designation
	}
	if err := u.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}
	return user.Public(), nil
}

func (u *authUsecase) ChangePassword(ctx context.Context, userID uuid.UUID, dto ChangePasswordDTO) error {
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if !hash.CheckPasswordHash(dto.CurrentPassword, user.PasswordHash) {
		return appErrors.ErrInvalidCredentials
	}
	hashed, err := hash.HashPassword(dto.NewPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hashed
	return u.userRepo.Update(ctx, user)
}

func (u *authUsecase) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	if refreshToken == "" {
		return nil, appErrors.ErrInvalidRefreshToken
	}
	claims, err := u.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, appErrors.ErrInvalidRefreshToken
	}
	ok, err := u.userRepo.RefreshTokenExists(ctx, utils.HashToken(refreshToken))
	if err != nil || !ok {
		return nil, appErrors.ErrInvalidRefreshToken
	}
	user, err := u.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	_ = u.userRepo.DeleteRefreshToken(ctx, utils.HashToken(refreshToken))
	return u.issueTokens(ctx, user)
}

func (u *authUsecase) ForgotPassword(ctx context.Context, email string) (string, error) {
	user, err := u.userRepo.GetByEmail(ctx, strings.ToLower(email))
	if err != nil {
		return "", nil
	}
	tok := utils.RandomToken()
	if err := u.userRepo.SavePasswordReset(ctx, user.ID, utils.HashToken(tok), time.Now().Add(2*time.Hour)); err != nil {
		return "", err
	}
	return tok, nil
}

func (u *authUsecase) ResetPassword(ctx context.Context, dto ResetPasswordDTO) error {
	userID, err := u.userRepo.ConsumePasswordReset(ctx, utils.HashToken(dto.Token))
	if err != nil {
		return err
	}
	user, err := u.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	hashed, err := hash.HashPassword(dto.NewPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hashed
	return u.userRepo.Update(ctx, user)
}

func (u *authUsecase) VerifyEmail(ctx context.Context, token string) error {
	userID, err := u.userRepo.ConsumeEmailVerify(ctx, utils.HashToken(token))
	if err != nil {
		return err
	}
	return u.userRepo.MarkEmailVerified(ctx, userID)
}

func (u *authUsecase) GetInstructorPublic(ctx context.Context, id uuid.UUID) (*InstructorPublic, error) {
	return u.userRepo.GetInstructorPublic(ctx, id)
}
