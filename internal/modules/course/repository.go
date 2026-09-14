// internal/modules/course/repository.go
package course

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
)

type CourseRepository interface {
	ListCourses(ctx context.Context, search, category string, page, limit int) ([]Course, int, error)
	GetBySlug(ctx context.Context, slug string) (*Course, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Course, error)
	CreateCourse(ctx context.Context, course *Course) error
	UpdateCourse(ctx context.Context, course *Course) error
	SetPublished(ctx context.Context, id uuid.UUID, isPublished bool) error
	CountTotal(ctx context.Context) (int, error)
	CountByInstructor(ctx context.Context, instructorID uuid.UUID) (int, error)
	GetCategories(ctx context.Context) ([]Category, error)
	GetModulesByCourseID(ctx context.Context, courseID uuid.UUID) ([]Module, error)
}

type postgresCourseRepository struct {
	db *sql.DB
}

func NewCourseRepository(db *sql.DB) CourseRepository {
	return &postgresCourseRepository{db: db}
}

func (r *postgresCourseRepository) ListCourses(ctx context.Context, search, category string, page, limit int) ([]Course, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	whereClause := "WHERE c.deleted_at IS NULL AND c.is_published = true"
	args := []interface{}{}
	argID := 1

	if search != "" {
		whereClause += fmt.Sprintf(" AND (c.title ILIKE $%d OR c.subtitle ILIKE $%d)", argID, argID)
		args = append(args, "%"+search+"%")
		argID++
	}

	if category != "" {
		whereClause += fmt.Sprintf(" AND cat.slug = $%d", argID)
		args = append(args, category)
		argID++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM courses c 
		LEFT JOIN categories cat ON c.category_id = cat.id 
		%s`, whereClause)

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT c.id, c.title, c.slug, c.subtitle, c.description, c.category_id, COALESCE(cat.title, ''),
		       c.instructor_id, u.first_name || ' ' || u.last_name AS instructor_name, u.avatar, u.role,
		       c.thumbnail, c.price, c.discount_price, c.is_published, COALESCE(c.learning_points, '[]'::jsonb),
		       c.created_at, c.updated_at
		FROM courses c
		LEFT JOIN categories cat ON c.category_id = cat.id
		JOIN users u ON c.instructor_id = u.id
		%s
		ORDER BY c.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argID, argID+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var courses []Course
	for rows.Next() {
		var c Course
		var instructor CourseInstructorInfo
		var learningPointsJSON []byte

		err := rows.Scan(
			&c.ID, &c.Title, &c.Slug, &c.Subtitle, &c.Description, &c.CategoryID, &c.CategoryName,
			&c.InstructorID, &instructor.Name, &instructor.Avatar, &instructor.Role,
			&c.Thumbnail, &c.Price, &c.DiscountPrice, &c.IsPublished, &learningPointsJSON,
			&c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		instructor.ID = c.InstructorID
		c.Instructor = &instructor

		_ = json.Unmarshal(learningPointsJSON, &c.LearningPoints)
		if c.LearningPoints == nil {
			c.LearningPoints = []string{}
		}

		courses = append(courses, c)
	}

	return courses, total, nil
}

func (r *postgresCourseRepository) GetBySlug(ctx context.Context, slug string) (*Course, error) {
	query := `
		SELECT c.id, c.title, c.slug, c.subtitle, c.description, c.category_id, COALESCE(cat.title, ''),
		       c.instructor_id, u.first_name || ' ' || u.last_name AS instructor_name, u.avatar, u.role,
		       c.thumbnail, c.price, c.discount_price, c.is_published, COALESCE(c.learning_points, '[]'::jsonb),
		       c.created_at, c.updated_at
		FROM courses c
		LEFT JOIN categories cat ON c.category_id = cat.id
		JOIN users u ON c.instructor_id = u.id
		WHERE c.slug = $1 AND c.deleted_at IS NULL
	`
	var c Course
	var instructor CourseInstructorInfo
	var learningPointsJSON []byte

	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&c.ID, &c.Title, &c.Slug, &c.Subtitle, &c.Description, &c.CategoryID, &c.CategoryName,
		&c.InstructorID, &instructor.Name, &instructor.Avatar, &instructor.Role,
		&c.Thumbnail, &c.Price, &c.DiscountPrice, &c.IsPublished, &learningPointsJSON,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.ErrCourseNotFound
		}
		return nil, err
	}

	instructor.ID = c.InstructorID
	c.Instructor = &instructor
	_ = json.Unmarshal(learningPointsJSON, &c.LearningPoints)
	if c.LearningPoints == nil {
		c.LearningPoints = []string{}
	}

	modules, err := r.GetModulesByCourseID(ctx, c.ID)
	if err == nil {
		c.Modules = modules
	}

	return &c, nil
}

func (r *postgresCourseRepository) GetByID(ctx context.Context, id uuid.UUID) (*Course, error) {
	query := `
		SELECT c.id, c.title, c.slug, c.subtitle, c.description, c.category_id, COALESCE(cat.title, ''),
		       c.instructor_id, u.first_name || ' ' || u.last_name AS instructor_name, u.avatar, u.role,
		       c.thumbnail, c.price, c.discount_price, c.is_published, COALESCE(c.learning_points, '[]'::jsonb),
		       c.created_at, c.updated_at
		FROM courses c
		LEFT JOIN categories cat ON c.category_id = cat.id
		JOIN users u ON c.instructor_id = u.id
		WHERE c.id = $1 AND c.deleted_at IS NULL
	`
	var c Course
	var instructor CourseInstructorInfo
	var learningPointsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Title, &c.Slug, &c.Subtitle, &c.Description, &c.CategoryID, &c.CategoryName,
		&c.InstructorID, &instructor.Name, &instructor.Avatar, &instructor.Role,
		&c.Thumbnail, &c.Price, &c.DiscountPrice, &c.IsPublished, &learningPointsJSON,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.ErrCourseNotFound
		}
		return nil, err
	}

	instructor.ID = c.InstructorID
	c.Instructor = &instructor
	_ = json.Unmarshal(learningPointsJSON, &c.LearningPoints)
	if c.LearningPoints == nil {
		c.LearningPoints = []string{}
	}

	modules, err := r.GetModulesByCourseID(ctx, c.ID)
	if err == nil {
		c.Modules = modules
	}

	return &c, nil
}

func (r *postgresCourseRepository) CreateCourse(ctx context.Context, course *Course) error {
	query := `
		INSERT INTO courses (id, title, slug, subtitle, description, category_id, instructor_id, thumbnail, price, discount_price, is_published, learning_points)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at, updated_at
	`
	if course.ID == uuid.Nil {
		course.ID = uuid.New()
	}

	learningJSON, _ := json.Marshal(course.LearningPoints)

	return r.db.QueryRowContext(
		ctx, query,
		course.ID, course.Title, course.Slug, course.Subtitle, course.Description,
		course.CategoryID, course.InstructorID, course.Thumbnail, course.Price, course.DiscountPrice,
		course.IsPublished, learningJSON,
	).Scan(&course.CreatedAt, &course.UpdatedAt)
}

func (r *postgresCourseRepository) UpdateCourse(ctx context.Context, course *Course) error {
	query := `
		UPDATE courses 
		SET title = $1, subtitle = $2, description = $3, category_id = $4, thumbnail = $5, 
		    price = $6, discount_price = $7, learning_points = $8, updated_at = CURRENT_TIMESTAMP
		WHERE id = $9 AND deleted_at IS NULL
	`
	learningJSON, _ := json.Marshal(course.LearningPoints)
	_, err := r.db.ExecContext(
		ctx, query,
		course.Title, course.Subtitle, course.Description, course.CategoryID, course.Thumbnail,
		course.Price, course.DiscountPrice, learningJSON, course.ID,
	)
	return err
}

func (r *postgresCourseRepository) SetPublished(ctx context.Context, id uuid.UUID, isPublished bool) error {
	query := `UPDATE courses SET is_published = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, query, isPublished, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return appErrors.ErrCourseNotFound
	}
	return nil
}

func (r *postgresCourseRepository) CountTotal(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM courses WHERE deleted_at IS NULL`
	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (r *postgresCourseRepository) CountByInstructor(ctx context.Context, instructorID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM courses WHERE instructor_id = $1 AND deleted_at IS NULL`
	var count int
	err := r.db.QueryRowContext(ctx, query, instructorID).Scan(&count)
	return count, err
}

func (r *postgresCourseRepository) GetCategories(ctx context.Context) ([]Category, error) {
	query := `SELECT id, title, slug, thumbnail, created_at FROM categories ORDER BY title ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Title, &cat.Slug, &cat.Thumbnail, &cat.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

func (r *postgresCourseRepository) GetModulesByCourseID(ctx context.Context, courseID uuid.UUID) ([]Module, error) {
	mQuery := `
		SELECT id, course_id, title, description, position, is_published, created_at
		FROM modules
		WHERE course_id = $1 AND is_published = true
		ORDER BY position ASC
	`
	mRows, err := r.db.QueryContext(ctx, mQuery, courseID)
	if err != nil {
		return nil, err
	}
	defer mRows.Close()

	var modules []Module
	for mRows.Next() {
		var m Module
		if err := mRows.Scan(&m.ID, &m.CourseID, &m.Title, &m.Description, &m.Position, &m.IsPublished, &m.CreatedAt); err != nil {
			return nil, err
		}

		lQuery := `
			SELECT id, module_id, title, description, video_url, duration, is_free, is_published, position, COALESCE(resources, '[]'::jsonb), created_at
			FROM lessons
			WHERE module_id = $1 AND is_published = true
			ORDER BY position ASC
		`
		lRows, err := r.db.QueryContext(ctx, lQuery, m.ID)
		if err == nil {
			var lessons []Lesson
			for lRows.Next() {
				var l Lesson
				var resourcesJSON []byte
				if err := lRows.Scan(
					&l.ID, &l.ModuleID, &l.Title, &l.Description, &l.VideoURL, &l.Duration,
					&l.IsFree, &l.IsPublished, &l.Position, &resourcesJSON, &l.CreatedAt,
				); err == nil {
					_ = json.Unmarshal(resourcesJSON, &l.Resources)
					if l.Resources == nil {
						l.Resources = []LessonResource{}
					}
					lessons = append(lessons, l)
				}
			}
			lRows.Close()
			m.Lessons = lessons
		}
		modules = append(modules, m)
	}
	return modules, nil
}
