// internal/modules/auth/models.go
package auth

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleStudent    UserRole = "student"
	RoleInstructor UserRole = "instructor"
	RoleAdmin      UserRole = "admin"
)

type UserStatus string

const (
	StatusActive    UserStatus = "active"
	StatusSuspended UserStatus = "suspended"
	StatusPending   UserStatus = "pending"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	FirstName    string     `json:"firstName"`
	LastName     string     `json:"lastName"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         UserRole   `json:"role"`
	Status       UserStatus `json:"status"`
	Avatar       *string    `json:"avatar"`
	Bio          *string    `json:"bio"`
	Occupation   *string    `json:"occupation"`
	Phone        *string    `json:"phone"`
	Website      *string    `json:"website"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"-"`
}

type RegisterDTO struct {
	FirstName string   `json:"firstName" binding:"required"`
	LastName  string   `json:"lastName" binding:"required"`
	Email     string   `json:"email" binding:"required,email"`
	Password  string   `json:"password" binding:"required,min=6"`
	Role      UserRole `json:"role"`
}

type LoginDTO struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

type UpdateProfileDTO struct {
	FirstName  *string `json:"firstName"`
	LastName   *string `json:"lastName"`
	Bio        *string `json:"bio"`
	Occupation *string `json:"occupation"`
	Phone      *string `json:"phone"`
	Website    *string `json:"website"`
	Avatar     *string `json:"avatar"`
}

type ChangePasswordDTO struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
}

type RefreshTokenDTO struct {
	RefreshToken string `json:"refreshToken"`
}

type ForgotPasswordDTO struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordDTO struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}
