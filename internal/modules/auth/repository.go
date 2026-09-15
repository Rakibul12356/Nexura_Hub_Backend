// internal/modules/auth/repository.go
package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status UserStatus) error
	CountByRole(ctx context.Context, role UserRole) (int, error)
}

type postgresUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation, phone, website)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	return r.db.QueryRowContext(
		ctx, query,
		user.ID, user.FirstName, user.LastName, user.Email, user.PasswordHash,
		user.Role, user.Status, user.Avatar, user.Bio, user.Occupation, user.Phone, user.Website,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation, phone, website, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	var u User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash,
		&u.Role, &u.Status, &u.Avatar, &u.Bio, &u.Occupation, &u.Phone, &u.Website,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation, phone, website, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	var u User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash,
		&u.Role, &u.Status, &u.Avatar, &u.Bio, &u.Occupation, &u.Phone, &u.Website,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *postgresUserRepository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, password_hash = $3, role = $4, status = $5, avatar = $6, bio = $7, occupation = $8, phone = $9, website = $10, updated_at = CURRENT_TIMESTAMP
		WHERE id = $11 AND deleted_at IS NULL
	`
	res, err := r.db.ExecContext(
		ctx, query,
		user.FirstName, user.LastName, user.PasswordHash, user.Role, user.Status,
		user.Avatar, user.Bio, user.Occupation, user.Phone, user.Website, user.ID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return appErrors.ErrUserNotFound
	}
	return nil
}

func (r *postgresUserRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status UserStatus) error {
	query := `UPDATE users SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return appErrors.ErrUserNotFound
	}
	return nil
}

func (r *postgresUserRepository) CountByRole(ctx context.Context, role UserRole) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE role = $1 AND deleted_at IS NULL`
	var count int
	err := r.db.QueryRowContext(ctx, query, role).Scan(&count)
	return count, err
}
