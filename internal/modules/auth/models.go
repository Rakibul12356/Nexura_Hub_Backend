package auth

import (
	"strings"
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
	ID              uuid.UUID  `json:"id"`
	FirstName       string     `json:"firstName"`
	LastName        string     `json:"lastName"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	Role            UserRole   `json:"role"`
	Status          UserStatus `json:"status,omitempty"`
	Avatar          *string    `json:"avatar"`
	Bio             *string    `json:"bio"`
	Occupation      *string    `json:"occupation"`
	Phone           *string    `json:"phone"`
	Website         *string    `json:"website"`
	Designation     *string    `json:"designation,omitempty"`
	EmailVerifiedAt *time.Time `json:"emailVerifiedAt,omitempty"`
	LastSeenAt      *time.Time `json:"lastSeenAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt,omitempty"`
	UpdatedAt       time.Time  `json:"updatedAt,omitempty"`
	DeletedAt       *time.Time `json:"-"`
}

func (u *User) Public() *User {
	if u == nil {
		return nil
	}
	cp := *u
	cp.PasswordHash = ""
	return &cp
}

type RegisterDTO struct {
	FirstName       string   `json:"firstName" binding:"required"`
	LastName        string   `json:"lastName" binding:"required"`
	Email           string   `json:"email" binding:"required,email"`
	Password        string   `json:"password" binding:"required,min=6"`
	ConfirmPassword string   `json:"confirmPassword"`
	Role            UserRole `json:"role"`
}

type LoginDTO struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=6"`
	RememberMe bool   `json:"rememberMe"`
}

type AuthResponse struct {
	User         *User  `json:"user"`
	Token        string `json:"token"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken"`
}

type TokenPair struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

type UpdateProfileDTO struct {
	FirstName  *string `json:"firstName"`
	LastName   *string `json:"lastName"`
	Email      *string `json:"email"`
	Bio        *string `json:"bio"`
	Occupation *string `json:"occupation"`
	Phone      *string `json:"phone"`
	Website    *string `json:"website"`
	Avatar     *string `json:"avatar"`
	Designation *string `json:"designation"`
}

type ChangePasswordDTO struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=6"`
}

type RefreshTokenDTO struct {
	RefreshToken     string `json:"refreshToken"`
	RefreshTokenAlt  string `json:"refresh_token"`
}

func (d RefreshTokenDTO) Token() string {
	if strings.TrimSpace(d.RefreshToken) != "" {
		return strings.TrimSpace(d.RefreshToken)
	}
	return strings.TrimSpace(d.RefreshTokenAlt)
}

type ForgotPasswordDTO struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordDTO struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}

type VerifyEmailDTO struct {
	Token string `json:"token" binding:"required"`
}

type InstructorPublic struct {
	ID            uuid.UUID `json:"id"`
	Name          string    `json:"name"`
	Designation   string    `json:"designation"`
	Avatar        *string   `json:"avatar"`
	Bio           *string   `json:"bio"`
	CoursesCount  int       `json:"coursesCount"`
	StudentsCount int       `json:"studentsCount"`
	ReviewsCount  int       `json:"reviewsCount"`
	Rating        float64   `json:"rating"`
	Courses       any       `json:"courses,omitempty"`
}
