package auth

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status UserStatus) error
	UpdateRole(ctx context.Context, id uuid.UUID, role UserRole) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	CountByRole(ctx context.Context, role UserRole) (int, error)
	CountAdmins(ctx context.Context) (int, error)
	ListUsers(ctx context.Context, role, search string, page, limit int) ([]ManagedUserRow, int, error)
	SaveRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	DeleteRefreshToken(ctx context.Context, tokenHash string) error
	DeleteUserRefreshTokens(ctx context.Context, userID uuid.UUID) error
	RefreshTokenExists(ctx context.Context, tokenHash string) (bool, error)
	SavePasswordReset(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	ConsumePasswordReset(ctx context.Context, tokenHash string) (uuid.UUID, error)
	SaveEmailVerify(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	ConsumeEmailVerify(ctx context.Context, tokenHash string) (uuid.UUID, error)
	MarkEmailVerified(ctx context.Context, userID uuid.UUID) error
	EnsureWallet(ctx context.Context, ownerType string, userID uuid.UUID) error
	GetInstructorPublic(ctx context.Context, id uuid.UUID) (*InstructorPublic, error)
}

type ManagedUserRow struct {
	User
	JoinDate                  time.Time
	EnrolledCoursesCount      int
	TotalSpent                float64
	CreatedCoursesCount       int
	TotalStudentsCount        int
	TotalEarnings             float64
	AdminCommissionGenerated  float64
}

type postgresUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation, phone, website, designation)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at
	`
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	return r.db.QueryRowContext(
		ctx, query,
		user.ID, user.FirstName, user.LastName, user.Email, user.PasswordHash,
		user.Role, user.Status, user.Avatar, user.Bio, user.Occupation, user.Phone, user.Website, user.Designation,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

func scanUser(row interface{ Scan(dest ...any) error }) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.PasswordHash,
		&u.Role, &u.Status, &u.Avatar, &u.Bio, &u.Occupation, &u.Phone, &u.Website, &u.Designation,
		&u.EmailVerifiedAt, &u.LastSeenAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

const userSelect = `
	SELECT id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation, phone, website,
	       designation, email_verified_at, last_seen_at, created_at, updated_at
	FROM users
`

func (r *postgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return scanUser(r.db.QueryRowContext(ctx, userSelect+` WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	return scanUser(r.db.QueryRowContext(ctx, userSelect+` WHERE email = $1 AND deleted_at IS NULL`, email))
}

func (r *postgresUserRepository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET first_name = $1, last_name = $2, email = $3, password_hash = $4, role = $5, status = $6,
		    avatar = $7, bio = $8, occupation = $9, phone = $10, website = $11, designation = $12,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $13 AND deleted_at IS NULL
	`
	res, err := r.db.ExecContext(
		ctx, query,
		user.FirstName, user.LastName, user.Email, user.PasswordHash, user.Role, user.Status,
		user.Avatar, user.Bio, user.Occupation, user.Phone, user.Website, user.Designation, user.ID,
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
	res, err := r.db.ExecContext(ctx, `UPDATE users SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND deleted_at IS NULL`, status, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return appErrors.ErrUserNotFound
	}
	return nil
}

func (r *postgresUserRepository) UpdateRole(ctx context.Context, id uuid.UUID, role UserRole) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET role = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND deleted_at IS NULL`, role, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return appErrors.ErrUserNotFound
	}
	return nil
}

func (r *postgresUserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET deleted_at = CURRENT_TIMESTAMP, status = 'suspended', updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return appErrors.ErrUserNotFound
	}
	return nil
}

func (r *postgresUserRepository) CountByRole(ctx context.Context, role UserRole) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = $1 AND deleted_at IS NULL`, role).Scan(&count)
	return count, err
}

func (r *postgresUserRepository) CountAdmins(ctx context.Context) (int, error) {
	return r.CountByRole(ctx, RoleAdmin)
}

func (r *postgresUserRepository) ListUsers(ctx context.Context, role, search string, page, limit int) ([]ManagedUserRow, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit
	where := `WHERE u.deleted_at IS NULL`
	args := []any{}
	i := 1
	if role != "" {
		where += ` AND u.role = $` + strconv.Itoa(i)
		args = append(args, role)
		i++
	}
	if search != "" {
		where += ` AND (u.first_name ILIKE $` + strconv.Itoa(i) + ` OR u.last_name ILIKE $` + strconv.Itoa(i) + ` OR u.email ILIKE $` + strconv.Itoa(i) + `)`
		args = append(args, "%"+search+"%")
		i++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users u `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT u.id, u.first_name, u.last_name, u.email, u.password_hash, u.role, u.status, u.avatar, u.bio,
		       u.occupation, u.phone, u.website, u.designation, u.email_verified_at, u.last_seen_at, u.created_at, u.updated_at,
		       COALESCE((SELECT COUNT(*) FROM enrollments e WHERE e.user_id = u.id), 0),
		       COALESCE((SELECT SUM(p.amount_paid) FROM payments p WHERE p.user_id = u.id AND p.status = 'paid'), 0),
		       COALESCE((SELECT COUNT(*) FROM courses c WHERE c.instructor_id = u.id AND c.deleted_at IS NULL), 0),
		       COALESCE((SELECT COUNT(DISTINCT e.user_id) FROM enrollments e JOIN courses c ON c.id = e.course_id WHERE c.instructor_id = u.id), 0),
		       COALESCE((SELECT SUM(t.instructor_earnings) FROM transactions t WHERE t.instructor_id = u.id AND t.status = 'completed'), 0),
		       COALESCE((SELECT SUM(t.admin_commission_amount) FROM transactions t WHERE t.instructor_id = u.id AND t.status = 'completed'), 0)
		FROM users u
	` + where + ` ORDER BY u.created_at DESC LIMIT $` + strconv.Itoa(i) + ` OFFSET $` + strconv.Itoa(i+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []ManagedUserRow
	for rows.Next() {
		var m ManagedUserRow
		if err := rows.Scan(
			&m.ID, &m.FirstName, &m.LastName, &m.Email, &m.PasswordHash, &m.Role, &m.Status, &m.Avatar, &m.Bio,
			&m.Occupation, &m.Phone, &m.Website, &m.Designation, &m.EmailVerifiedAt, &m.LastSeenAt, &m.CreatedAt, &m.UpdatedAt,
			&m.EnrolledCoursesCount, &m.TotalSpent, &m.CreatedCoursesCount, &m.TotalStudentsCount, &m.TotalEarnings, &m.AdminCommissionGenerated,
		); err != nil {
			return nil, 0, err
		}
		m.JoinDate = m.CreatedAt
		out = append(out, m)
	}
	if out == nil {
		out = []ManagedUserRow{}
	}
	return out, total, nil
}

func (r *postgresUserRepository) SaveRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES ($1,$2,$3,$4)`, uuid.New(), userID, tokenHash, expiresAt)
	return err
}

func (r *postgresUserRepository) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}

func (r *postgresUserRepository) DeleteUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}

func (r *postgresUserRepository) RefreshTokenExists(ctx context.Context, tokenHash string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM refresh_tokens WHERE token_hash = $1 AND expires_at > NOW())`, tokenHash).Scan(&exists)
	return exists, err
}

func (r *postgresUserRepository) SavePasswordReset(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, _ = r.db.ExecContext(ctx, `DELETE FROM password_resets WHERE user_id = $1`, userID)
	_, err := r.db.ExecContext(ctx, `INSERT INTO password_resets (id, user_id, token_hash, expires_at) VALUES ($1,$2,$3,$4)`, uuid.New(), userID, tokenHash, expiresAt)
	return err
}

func (r *postgresUserRepository) ConsumePasswordReset(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `DELETE FROM password_resets WHERE token_hash = $1 AND expires_at > NOW() RETURNING user_id`, tokenHash).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, appErrors.ErrInvalidResetToken
	}
	return userID, err
}

func (r *postgresUserRepository) SaveEmailVerify(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, _ = r.db.ExecContext(ctx, `DELETE FROM email_verifications WHERE user_id = $1`, userID)
	_, err := r.db.ExecContext(ctx, `INSERT INTO email_verifications (id, user_id, token_hash, expires_at) VALUES ($1,$2,$3,$4)`, uuid.New(), userID, tokenHash, expiresAt)
	return err
}

func (r *postgresUserRepository) ConsumeEmailVerify(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.QueryRowContext(ctx, `DELETE FROM email_verifications WHERE token_hash = $1 AND expires_at > NOW() RETURNING user_id`, tokenHash).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, appErrors.ErrInvalidVerifyToken
	}
	return userID, err
}

func (r *postgresUserRepository) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET email_verified_at = NOW(), updated_at = NOW() WHERE id = $1`, userID)
	return err
}

func (r *postgresUserRepository) EnsureWallet(ctx context.Context, ownerType string, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO wallets (id, owner_type, user_id, balance, currency)
		VALUES ($1, $2, $3, 0, 'BDT')
		ON CONFLICT (owner_type, user_id) DO NOTHING
	`, uuid.New(), ownerType, userID)
	return err
}

func (r *postgresUserRepository) GetInstructorPublic(ctx context.Context, id uuid.UUID) (*InstructorPublic, error) {
	u, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u.Role != RoleInstructor && u.Role != RoleAdmin {
		return nil, appErrors.ErrUserNotFound
	}
	p := &InstructorPublic{
		ID:   u.ID,
		Name: u.FirstName + " " + u.LastName,
		Avatar: u.Avatar,
		Bio:    u.Bio,
	}
	if u.Designation != nil {
		p.Designation = *u.Designation
	} else if u.Occupation != nil {
		p.Designation = *u.Occupation
	}
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM courses WHERE instructor_id = $1 AND deleted_at IS NULL AND is_published = true`, id).Scan(&p.CoursesCount)
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT e.user_id) FROM enrollments e JOIN courses c ON c.id = e.course_id WHERE c.instructor_id = $1`, id).Scan(&p.StudentsCount)
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(AVG(r.rating),0) FROM reviews r JOIN courses c ON c.id = r.course_id WHERE c.instructor_id = $1`, id).Scan(&p.ReviewsCount, &p.Rating)
	return p, nil
}

