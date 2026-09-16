package course

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/pkg/utils"
)

type CourseRepository interface {
	ListCourses(ctx context.Context, f ListFilter) ([]Course, int, error)
	GetByIDOrSlug(ctx context.Context, idOrSlug string) (*Course, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Course, error)
	CreateCourse(ctx context.Context, course *Course) error
	UpdateCourse(ctx context.Context, course *Course) error
	SoftDeleteCourse(ctx context.Context, id uuid.UUID) error
	SetPublished(ctx context.Context, id uuid.UUID, published bool) error
	SetFeatured(ctx context.Context, id uuid.UUID, featured bool) error
	GetCategories(ctx context.Context) ([]Category, error)
	GetCategory(ctx context.Context, id uuid.UUID) (*Category, error)
	CreateCategory(ctx context.Context, cat *Category) error
	UpdateCategory(ctx context.Context, cat *Category) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error
	FindCategoryID(ctx context.Context, id *uuid.UUID, slugOrTitle string) (*uuid.UUID, string, error)
	GetModules(ctx context.Context, courseID uuid.UUID) ([]Module, error)
	GetModule(ctx context.Context, moduleID uuid.UUID) (*Module, error)
	AddModule(ctx context.Context, m *Module) error
	UpdateModule(ctx context.Context, m *Module) error
	DeleteModule(ctx context.Context, id uuid.UUID) error
	ReorderModules(ctx context.Context, courseID uuid.UUID, ids []uuid.UUID) error
	GetLesson(ctx context.Context, lessonID uuid.UUID) (*Lesson, error)
	AddLesson(ctx context.Context, l *Lesson) error
	UpdateLesson(ctx context.Context, l *Lesson) error
	DeleteLesson(ctx context.Context, id uuid.UUID) error
	ReorderLessons(ctx context.Context, moduleID uuid.UUID, ids []uuid.UUID) error
	AddResource(ctx context.Context, lessonID uuid.UUID, r *LessonResource) error
	DeleteResource(ctx context.Context, lessonID, resourceID uuid.UUID) error
	GetReviews(ctx context.Context, courseID uuid.UUID) ([]CourseReview, error)
	UpsertReview(ctx context.Context, courseID, userID uuid.UUID, rating int, comment string) (*CourseReview, error)
	UpdateReview(ctx context.Context, id, userID uuid.UUID, rating int, comment string, isAdmin bool) (*CourseReview, error)
	DeleteReview(ctx context.Context, id, userID uuid.UUID, isAdmin bool) error
	GetNotes(ctx context.Context, lessonID, userID uuid.UUID) ([]Note, error)
	AddNote(ctx context.Context, lessonID, userID uuid.UUID, timestamp int, text string) (*Note, error)
	DeleteNote(ctx context.Context, lessonID, noteID, userID uuid.UUID) error
	GetDiscussions(ctx context.Context, lessonID uuid.UUID) ([]Discussion, error)
	AddDiscussion(ctx context.Context, lessonID, courseID, userID uuid.UUID, content string) (*Discussion, error)
	AddReply(ctx context.Context, discussionID, userID uuid.UUID, content string) (*DiscussionReply, error)
	ToggleUpvote(ctx context.Context, discussionID, userID uuid.UUID) (int, error)
	IsEnrolled(ctx context.Context, userID, courseID uuid.UUID) (bool, error)
	CompletedLessonIDs(ctx context.Context, userID, courseID uuid.UUID) (map[uuid.UUID]bool, error)
	GetEnrollmentProgress(ctx context.Context, userID, courseID uuid.UUID) float64
	CourseIDForModule(ctx context.Context, moduleID uuid.UUID) (uuid.UUID, error)
	CourseIDForLesson(ctx context.Context, lessonID uuid.UUID) (uuid.UUID, uuid.UUID, error)
	OwnerIDForCourse(ctx context.Context, courseID uuid.UUID) (uuid.UUID, error)
	ListInstructorCourses(ctx context.Context, instructorID uuid.UUID, isAdmin bool) ([]Course, error)
	GetEnrolledCourses(ctx context.Context, userID uuid.UUID) ([]EnrolledCourse, error)
}

type postgresCourseRepository struct {
	db *sql.DB
}

func NewCourseRepository(db *sql.DB) CourseRepository {
	return &postgresCourseRepository{db: db}
}

func (r *postgresCourseRepository) ListCourses(ctx context.Context, f ListFilter) ([]Course, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	where := []string{"c.deleted_at IS NULL"}
	args := []any{}
	i := 1

	publishedOnly := !f.IncludeUnpublished && f.ViewerRole != "admin" && f.MineUserID == nil
	if f.IsPublished != nil {
		where = append(where, "c.is_published = $"+strconv.Itoa(i))
		args = append(args, *f.IsPublished)
		i++
	} else if publishedOnly {
		where = append(where, "c.is_published = true")
	}
	if f.IsFeatured != nil {
		where = append(where, "c.is_featured = $"+strconv.Itoa(i))
		args = append(args, *f.IsFeatured)
		i++
	}
	if f.Search != "" {
		where = append(where, "(c.title ILIKE $"+strconv.Itoa(i)+" OR c.subtitle ILIKE $"+strconv.Itoa(i)+")")
		args = append(args, "%"+f.Search+"%")
		i++
	}
	if f.Category != "" {
		where = append(where, "(cat.slug = $"+strconv.Itoa(i)+" OR cat.title ILIKE $"+strconv.Itoa(i)+")")
		args = append(args, f.Category)
		i++
	}
	if f.Price == "free" {
		where = append(where, "c.price = 0")
	} else if f.Price == "paid" {
		where = append(where, "c.price > 0")
	}
	if f.InstructorID != nil {
		where = append(where, "c.instructor_id = $"+strconv.Itoa(i))
		args = append(args, *f.InstructorID)
		i++
	}
	if f.MineUserID != nil && f.ViewerRole != "admin" {
		where = append(where, "c.instructor_id = $"+strconv.Itoa(i))
		args = append(args, *f.MineUserID)
		i++
	}
	if f.CreatorType != "" {
		where = append(where, "c.creator_type = $"+strconv.Itoa(i))
		args = append(args, f.CreatorType)
		i++
	}

	w := "WHERE " + strings.Join(where, " AND ")
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM courses c LEFT JOIN categories cat ON c.category_id = cat.id `+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	order := "c.created_at DESC"
	switch f.Sort {
	case "price-asc":
		order = "c.price ASC"
	case "price-desc":
		order = "c.price DESC"
	case "popular":
		order = "(SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id) DESC, c.created_at DESC"
	case "createdAt:asc":
		order = "c.created_at ASC"
	}

	query := fmt.Sprintf(`
		SELECT c.id, c.slug, c.title, c.subtitle, c.description, c.category_id, COALESCE(cat.title,''),
		       c.instructor_id, c.creator_type, COALESCE(c.thumbnail,''), c.price, c.discount_price,
		       c.is_published, c.is_featured, COALESCE(c.learning_points, '{}'),
		       c.created_at, c.updated_at,
		       u.first_name, u.last_name, u.avatar, u.designation, u.bio, u.role,
		       (SELECT COUNT(*) FROM modules m WHERE m.course_id = c.id),
		       (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id),
		       COALESCE((SELECT SUM(t.price) FROM transactions t WHERE t.course_id = c.id AND t.status='completed'),0),
		       COALESCE((SELECT SUM(t.admin_commission_amount) FROM transactions t WHERE t.course_id = c.id AND t.status='completed'),0)
		FROM courses c
		LEFT JOIN categories cat ON c.category_id = cat.id
		JOIN users u ON u.id = c.instructor_id
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, w, order, i, i+1)
	args = append(args, f.Limit, utils.Offset(f.Page, f.Limit))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var courses []Course
	for rows.Next() {
		c, err := scanCourseListRow(rows)
		if err != nil {
			return nil, 0, err
		}
		courses = append(courses, *c)
	}
	if courses == nil {
		courses = []Course{}
	}
	return courses, total, nil
}

func scanCourseListRow(rows *sql.Rows) (*Course, error) {
	var c Course
	var catID *uuid.UUID
	var first, last string
	var avatar, designation, bio *string
	var role string
	var points pq.StringArray
	err := rows.Scan(
		&c.ID, &c.Slug, &c.Title, &c.Subtitle, &c.Description, &catID, &c.Category,
		&c.InstructorID, &c.CreatorType, &c.Thumbnail, &c.Price, &c.DiscountPrice,
		&c.IsPublished, &c.IsFeatured, &points, &c.CreatedAt, &c.UpdatedAt,
		&first, &last, &avatar, &designation, &bio, &role,
		&c.TotalChapters, &c.EnrollmentsCount, &c.TotalRevenue, &c.AdminEarnings,
	)
	if err != nil {
		return nil, err
	}
	c.CategoryID = catID
	c.Image = c.Thumbnail
	c.LearningPoints = []string(points)
	if c.LearningPoints == nil {
		c.LearningPoints = []string{}
	}
	des := ""
	if designation != nil {
		des = *designation
	}
	c.Instructor = &CourseInstructorInfo{
		ID: c.InstructorID, Name: strings.TrimSpace(first + " " + last),
		Designation: des, Avatar: avatar, Bio: bio, Role: role,
	}
	return &c, nil
}

func (r *postgresCourseRepository) GetByID(ctx context.Context, id uuid.UUID) (*Course, error) {
	return r.GetByIDOrSlug(ctx, id.String())
}

func (r *postgresCourseRepository) GetByIDOrSlug(ctx context.Context, idOrSlug string) (*Course, error) {
	query := `
		SELECT c.id, c.slug, c.title, c.subtitle, c.description, c.category_id, COALESCE(cat.title,''),
		       c.instructor_id, c.creator_type, COALESCE(c.thumbnail,''), c.price, c.discount_price,
		       c.is_published, c.is_featured, COALESCE(c.learning_points, '{}'),
		       c.created_at, c.updated_at,
		       u.first_name, u.last_name, u.avatar, u.designation, u.bio, u.role
		FROM courses c
		LEFT JOIN categories cat ON c.category_id = cat.id
		JOIN users u ON u.id = c.instructor_id
		WHERE c.deleted_at IS NULL AND (c.id::text = $1 OR c.slug = $1)
	`
	var c Course
	var catID *uuid.UUID
	var first, last, role string
	var avatar, designation, bio *string
	var points pq.StringArray
	err := r.db.QueryRowContext(ctx, query, idOrSlug).Scan(
		&c.ID, &c.Slug, &c.Title, &c.Subtitle, &c.Description, &catID, &c.Category,
		&c.InstructorID, &c.CreatorType, &c.Thumbnail, &c.Price, &c.DiscountPrice,
		&c.IsPublished, &c.IsFeatured, &points, &c.CreatedAt, &c.UpdatedAt,
		&first, &last, &avatar, &designation, &bio, &role,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.ErrCourseNotFound
		}
		return nil, err
	}
	c.CategoryID = catID
	c.Image = c.Thumbnail
	c.LearningPoints = []string(points)
	if c.LearningPoints == nil {
		c.LearningPoints = []string{}
	}
	des := ""
	if designation != nil {
		des = *designation
	}
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM courses WHERE instructor_id=$1 AND deleted_at IS NULL AND is_published=true`, c.InstructorID).Scan(&c.TotalChapters)
	var coursesCount, studentsCount, reviewsCount int
	var rating float64
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM courses WHERE instructor_id=$1 AND deleted_at IS NULL AND is_published=true`, c.InstructorID).Scan(&coursesCount)
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT e.user_id) FROM enrollments e JOIN courses x ON x.id=e.course_id WHERE x.instructor_id=$1`, c.InstructorID).Scan(&studentsCount)
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(AVG(r.rating),0) FROM reviews r JOIN courses x ON x.id=r.course_id WHERE x.instructor_id=$1`, c.InstructorID).Scan(&reviewsCount, &rating)
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM modules WHERE course_id=$1`, c.ID).Scan(&c.TotalChapters)
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM enrollments WHERE course_id=$1`, c.ID).Scan(&c.EnrollmentsCount)
	c.Instructor = &CourseInstructorInfo{
		ID: c.InstructorID, Name: strings.TrimSpace(first + " " + last), Designation: des,
		Avatar: avatar, Bio: bio, Role: role, CoursesCount: coursesCount, StudentsCount: studentsCount,
		ReviewsCount: reviewsCount, Rating: utils.RoundMoney(rating),
	}
	mods, _ := r.GetModules(ctx, c.ID)
	c.Modules = mods
	revs, _ := r.GetReviews(ctx, c.ID)
	c.Reviews = revs
	return &c, nil
}

func (r *postgresCourseRepository) CreateCourse(ctx context.Context, course *Course) error {
	if course.ID == uuid.Nil {
		course.ID = uuid.New()
	}
	if course.LearningPoints == nil {
		course.LearningPoints = []string{}
	}
	q := `
		INSERT INTO courses (id, slug, title, subtitle, description, category_id, instructor_id, creator_type, thumbnail, price, discount_price, is_published, is_featured, learning_points)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, q,
		course.ID, course.Slug, course.Title, course.Subtitle, course.Description, course.CategoryID,
		course.InstructorID, course.CreatorType, course.Thumbnail, course.Price, course.DiscountPrice,
		course.IsPublished, course.IsFeatured, pq.Array(course.LearningPoints),
	).Scan(&course.CreatedAt, &course.UpdatedAt)
}

func (r *postgresCourseRepository) UpdateCourse(ctx context.Context, course *Course) error {
	q := `
		UPDATE courses SET title=$1, subtitle=$2, description=$3, category_id=$4, thumbnail=$5, price=$6,
		       discount_price=$7, is_published=$8, is_featured=$9, learning_points=$10, updated_at=NOW()
		WHERE id=$11 AND deleted_at IS NULL
	`
	res, err := r.db.ExecContext(ctx, q, course.Title, course.Subtitle, course.Description, course.CategoryID,
		course.Thumbnail, course.Price, course.DiscountPrice, course.IsPublished, course.IsFeatured,
		pq.Array(course.LearningPoints), course.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return appErrors.ErrCourseNotFound
	}
	return nil
}

func (r *postgresCourseRepository) SoftDeleteCourse(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `UPDATE courses SET deleted_at=NOW(), updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return appErrors.ErrCourseNotFound
	}
	return nil
}

func (r *postgresCourseRepository) SetPublished(ctx context.Context, id uuid.UUID, published bool) error {
	res, err := r.db.ExecContext(ctx, `UPDATE courses SET is_published=$1, updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL`, published, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return appErrors.ErrCourseNotFound
	}
	return nil
}

func (r *postgresCourseRepository) SetFeatured(ctx context.Context, id uuid.UUID, featured bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE courses SET is_featured=$1, updated_at=NOW() WHERE id=$2 AND deleted_at IS NULL`, featured, id)
	return err
}

func (r *postgresCourseRepository) GetCategories(ctx context.Context) ([]Category, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, title, slug, COALESCE(thumbnail,''), created_at FROM categories ORDER BY title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Title, &c.Slug, &c.Thumbnail, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Value, c.Label = c.Slug, c.Title
		out = append(out, c)
	}
	if out == nil {
		out = []Category{}
	}
	return out, nil
}

func (r *postgresCourseRepository) GetCategory(ctx context.Context, id uuid.UUID) (*Category, error) {
	var c Category
	err := r.db.QueryRowContext(ctx, `SELECT id, title, slug, COALESCE(thumbnail,''), created_at FROM categories WHERE id=$1`, id).
		Scan(&c.ID, &c.Title, &c.Slug, &c.Thumbnail, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrCategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Value, c.Label = c.Slug, c.Title
	return &c, nil
}

func (r *postgresCourseRepository) CreateCategory(ctx context.Context, cat *Category) error {
	if cat.ID == uuid.Nil {
		cat.ID = uuid.New()
	}
	return r.db.QueryRowContext(ctx, `INSERT INTO categories (id, title, slug, thumbnail) VALUES ($1,$2,$3,$4) RETURNING created_at`,
		cat.ID, cat.Title, cat.Slug, cat.Thumbnail).Scan(&cat.CreatedAt)
}

func (r *postgresCourseRepository) UpdateCategory(ctx context.Context, cat *Category) error {
	_, err := r.db.ExecContext(ctx, `UPDATE categories SET title=$1, slug=$2, thumbnail=$3 WHERE id=$4`, cat.Title, cat.Slug, cat.Thumbnail, cat.ID)
	return err
}

func (r *postgresCourseRepository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM categories WHERE id=$1`, id)
	return err
}

func (r *postgresCourseRepository) FindCategoryID(ctx context.Context, id *uuid.UUID, slugOrTitle string) (*uuid.UUID, string, error) {
	if id != nil {
		cat, err := r.GetCategory(ctx, *id)
		if err != nil {
			return id, "", err
		}
		return id, cat.Title, nil
	}
	if slugOrTitle == "" {
		return nil, "", nil
	}
	var found uuid.UUID
	var title string
	err := r.db.QueryRowContext(ctx, `SELECT id, title FROM categories WHERE slug=$1 OR title ILIKE $1 LIMIT 1`, slugOrTitle).Scan(&found, &title)
	if err != nil {
		return nil, "", nil
	}
	return &found, title, nil
}

func (r *postgresCourseRepository) GetModules(ctx context.Context, courseID uuid.UUID) ([]Module, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, course_id, title, description, position, is_published, created_at FROM modules WHERE course_id=$1 ORDER BY position, created_at`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mods []Module
	for rows.Next() {
		var m Module
		if err := rows.Scan(&m.ID, &m.CourseID, &m.Title, &m.Description, &m.Position, &m.IsPublished, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Lessons = []Lesson{}
		mods = append(mods, m)
	}
	if mods == nil {
		return []Module{}, nil
	}
	lrows, err := r.db.QueryContext(ctx, `
		SELECT l.id, l.module_id, l.title, l.description, l.video_url, l.duration, l.is_free, l.is_published, l.position, l.quiz_set_id, l.created_at,
		       qs.title
		FROM lessons l
		LEFT JOIN quiz_sets qs ON qs.id = l.quiz_set_id
		WHERE l.module_id IN (SELECT id FROM modules WHERE course_id=$1)
		ORDER BY l.position, l.created_at
	`, courseID)
	if err != nil {
		return mods, nil
	}
	defer lrows.Close()
	byMod := map[uuid.UUID][]Lesson{}
	for lrows.Next() {
		var l Lesson
		var quizTitle *string
		if err := lrows.Scan(&l.ID, &l.ModuleID, &l.Title, &l.Description, &l.VideoURL, &l.Duration, &l.IsFree, &l.IsPublished, &l.Position, &l.QuizSetID, &l.CreatedAt, &quizTitle); err != nil {
			return nil, err
		}
		l.QuizSetTitle = quizTitle
		l.HasQuiz = l.QuizSetID != nil
		l.Resources = r.resources(ctx, l.ID)
		byMod[l.ModuleID] = append(byMod[l.ModuleID], l)
	}
	for i := range mods {
		if ls, ok := byMod[mods[i].ID]; ok {
			mods[i].Lessons = ls
		}
	}
	return mods, nil
}

func (r *postgresCourseRepository) resources(ctx context.Context, lessonID uuid.UUID) []LessonResource {
	rows, err := r.db.QueryContext(ctx, `SELECT id, title, type, url, size, file_name, content FROM lesson_resources WHERE lesson_id=$1 ORDER BY created_at`, lessonID)
	if err != nil {
		return []LessonResource{}
	}
	defer rows.Close()
	var out []LessonResource
	for rows.Next() {
		var res LessonResource
		if err := rows.Scan(&res.ID, &res.Title, &res.Type, &res.URL, &res.Size, &res.FileName, &res.Content); err != nil {
			continue
		}
		out = append(out, res)
	}
	if out == nil {
		return []LessonResource{}
	}
	return out
}

func (r *postgresCourseRepository) GetModule(ctx context.Context, moduleID uuid.UUID) (*Module, error) {
	var m Module
	err := r.db.QueryRowContext(ctx, `SELECT id, course_id, title, description, position, is_published, created_at FROM modules WHERE id=$1`, moduleID).
		Scan(&m.ID, &m.CourseID, &m.Title, &m.Description, &m.Position, &m.IsPublished, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrModuleNotFound
	}
	if err != nil {
		return nil, err
	}
	mods, err := r.GetModules(ctx, m.CourseID)
	if err != nil {
		return &m, nil
	}
	for _, x := range mods {
		if x.ID == moduleID {
			return &x, nil
		}
	}
	m.Lessons = []Lesson{}
	return &m, nil
}

func (r *postgresCourseRepository) AddModule(ctx context.Context, m *Module) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	var pos int
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(position),0)+1 FROM modules WHERE course_id=$1`, m.CourseID).Scan(&pos)
	m.Position = pos
	return r.db.QueryRowContext(ctx, `INSERT INTO modules (id, course_id, title, description, position, is_published) VALUES ($1,$2,$3,$4,$5,$6) RETURNING created_at`,
		m.ID, m.CourseID, m.Title, m.Description, m.Position, m.IsPublished).Scan(&m.CreatedAt)
}

func (r *postgresCourseRepository) UpdateModule(ctx context.Context, m *Module) error {
	_, err := r.db.ExecContext(ctx, `UPDATE modules SET title=$1, description=$2, is_published=$3, updated_at=NOW() WHERE id=$4`, m.Title, m.Description, m.IsPublished, m.ID)
	return err
}

func (r *postgresCourseRepository) DeleteModule(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM modules WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return appErrors.ErrModuleNotFound
	}
	return nil
}

func (r *postgresCourseRepository) ReorderModules(ctx context.Context, courseID uuid.UUID, ids []uuid.UUID) error {
	for i, id := range ids {
		_, _ = r.db.ExecContext(ctx, `UPDATE modules SET position=$1, updated_at=NOW() WHERE id=$2 AND course_id=$3`, i+1, id, courseID)
	}
	return nil
}

func (r *postgresCourseRepository) GetLesson(ctx context.Context, lessonID uuid.UUID) (*Lesson, error) {
	var l Lesson
	var quizTitle *string
	err := r.db.QueryRowContext(ctx, `
		SELECT l.id, l.module_id, l.title, l.description, l.video_url, l.duration, l.is_free, l.is_published, l.position, l.quiz_set_id, l.created_at, qs.title
		FROM lessons l LEFT JOIN quiz_sets qs ON qs.id=l.quiz_set_id WHERE l.id=$1
	`, lessonID).Scan(&l.ID, &l.ModuleID, &l.Title, &l.Description, &l.VideoURL, &l.Duration, &l.IsFree, &l.IsPublished, &l.Position, &l.QuizSetID, &l.CreatedAt, &quizTitle)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrLessonNotFound
	}
	if err != nil {
		return nil, err
	}
	l.QuizSetTitle = quizTitle
	l.HasQuiz = l.QuizSetID != nil
	l.Resources = r.resources(ctx, l.ID)
	return &l, nil
}

func (r *postgresCourseRepository) AddLesson(ctx context.Context, l *Lesson) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	var pos int
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(position),0)+1 FROM lessons WHERE module_id=$1`, l.ModuleID).Scan(&pos)
	l.Position = pos
	return r.db.QueryRowContext(ctx, `
		INSERT INTO lessons (id, module_id, title, description, video_url, duration, is_free, is_published, position, quiz_set_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING created_at
	`, l.ID, l.ModuleID, l.Title, l.Description, l.VideoURL, l.Duration, l.IsFree, l.IsPublished, l.Position, l.QuizSetID).Scan(&l.CreatedAt)
}

func (r *postgresCourseRepository) UpdateLesson(ctx context.Context, l *Lesson) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE lessons SET title=$1, description=$2, video_url=$3, duration=$4, is_free=$5, is_published=$6, quiz_set_id=$7, updated_at=NOW()
		WHERE id=$8
	`, l.Title, l.Description, l.VideoURL, l.Duration, l.IsFree, l.IsPublished, l.QuizSetID, l.ID)
	return err
}

func (r *postgresCourseRepository) DeleteLesson(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM lessons WHERE id=$1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return appErrors.ErrLessonNotFound
	}
	return nil
}

func (r *postgresCourseRepository) ReorderLessons(ctx context.Context, moduleID uuid.UUID, ids []uuid.UUID) error {
	for i, id := range ids {
		_, _ = r.db.ExecContext(ctx, `UPDATE lessons SET position=$1, updated_at=NOW() WHERE id=$2 AND module_id=$3`, i+1, id, moduleID)
	}
	return nil
}

func (r *postgresCourseRepository) AddResource(ctx context.Context, lessonID uuid.UUID, res *LessonResource) error {
	if res.ID == uuid.Nil {
		res.ID = uuid.New()
	}
	if res.Type == "" {
		res.Type = "link"
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO lesson_resources (id, lesson_id, title, type, url, size, file_name, content) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		res.ID, lessonID, res.Title, res.Type, res.URL, res.Size, res.FileName, res.Content)
	return err
}

func (r *postgresCourseRepository) DeleteResource(ctx context.Context, lessonID, resourceID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM lesson_resources WHERE id=$1 AND lesson_id=$2`, resourceID, lessonID)
	return err
}

func (r *postgresCourseRepository) GetReviews(ctx context.Context, courseID uuid.UUID) ([]CourseReview, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.user_id, u.first_name || ' ' || u.last_name, u.avatar, r.rating, r.comment, r.created_at
		FROM reviews r JOIN users u ON u.id=r.user_id WHERE r.course_id=$1 ORDER BY r.created_at DESC
	`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CourseReview
	for rows.Next() {
		var rv CourseReview
		var created time.Time
		if err := rows.Scan(&rv.ID, &rv.UserID, &rv.StudentName, &rv.StudentAvatar, &rv.Rating, &rv.Comment, &created); err != nil {
			return nil, err
		}
		rv.Date = created.Format("2006-01-02")
		out = append(out, rv)
	}
	if out == nil {
		out = []CourseReview{}
	}
	return out, nil
}

func (r *postgresCourseRepository) UpsertReview(ctx context.Context, courseID, userID uuid.UUID, rating int, comment string) (*CourseReview, error) {
	id := uuid.New()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO reviews (id, course_id, user_id, rating, comment) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (course_id, user_id) DO UPDATE SET rating=EXCLUDED.rating, comment=EXCLUDED.comment
	`, id, courseID, userID, rating, comment)
	if err != nil {
		return nil, err
	}
	var rv CourseReview
	var created time.Time
	err = r.db.QueryRowContext(ctx, `
		SELECT r.id, r.user_id, u.first_name || ' ' || u.last_name, u.avatar, r.rating, r.comment, r.created_at
		FROM reviews r JOIN users u ON u.id=r.user_id WHERE r.course_id=$1 AND r.user_id=$2
	`, courseID, userID).Scan(&rv.ID, &rv.UserID, &rv.StudentName, &rv.StudentAvatar, &rv.Rating, &rv.Comment, &created)
	rv.Date = created.Format("2006-01-02")
	return &rv, err
}

func (r *postgresCourseRepository) UpdateReview(ctx context.Context, id, userID uuid.UUID, rating int, comment string, isAdmin bool) (*CourseReview, error) {
	q := `UPDATE reviews SET rating=$1, comment=$2 WHERE id=$3`
	args := []any{rating, comment, id}
	if !isAdmin {
		q += ` AND user_id=$4`
		args = append(args, userID)
	}
	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, appErrors.ErrReviewNotFound
	}
	var rv CourseReview
	var created time.Time
	err = r.db.QueryRowContext(ctx, `
		SELECT r.id, r.user_id, u.first_name || ' ' || u.last_name, u.avatar, r.rating, r.comment, r.created_at
		FROM reviews r JOIN users u ON u.id=r.user_id WHERE r.id=$1
	`, id).Scan(&rv.ID, &rv.UserID, &rv.StudentName, &rv.StudentAvatar, &rv.Rating, &rv.Comment, &created)
	rv.Date = created.Format("2006-01-02")
	return &rv, err
}

func (r *postgresCourseRepository) DeleteReview(ctx context.Context, id, userID uuid.UUID, isAdmin bool) error {
	q := `DELETE FROM reviews WHERE id=$1`
	args := []any{id}
	if !isAdmin {
		q += ` AND user_id=$2`
		args = append(args, userID)
	}
	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return appErrors.ErrReviewNotFound
	}
	return nil
}

func (r *postgresCourseRepository) GetNotes(ctx context.Context, lessonID, userID uuid.UUID) ([]Note, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, timestamp_sec, text, created_at FROM lesson_notes WHERE lesson_id=$1 AND user_id=$2 ORDER BY created_at`, lessonID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Note
	for rows.Next() {
		var n Note
		var created time.Time
		if err := rows.Scan(&n.ID, &n.Timestamp, &n.Text, &created); err != nil {
			return nil, err
		}
		n.CreatedAt = utils.DisplayTime(created)
		out = append(out, n)
	}
	if out == nil {
		out = []Note{}
	}
	return out, nil
}

func (r *postgresCourseRepository) AddNote(ctx context.Context, lessonID, userID uuid.UUID, timestamp int, text string) (*Note, error) {
	n := &Note{ID: uuid.New(), Timestamp: timestamp, Text: text}
	var created time.Time
	err := r.db.QueryRowContext(ctx, `INSERT INTO lesson_notes (id, user_id, lesson_id, timestamp_sec, text) VALUES ($1,$2,$3,$4,$5) RETURNING created_at`,
		n.ID, userID, lessonID, timestamp, text).Scan(&created)
	n.CreatedAt = utils.DisplayTime(created)
	return n, err
}

func (r *postgresCourseRepository) DeleteNote(ctx context.Context, lessonID, noteID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM lesson_notes WHERE id=$1 AND lesson_id=$2 AND user_id=$3`, noteID, lessonID, userID)
	return err
}

func (r *postgresCourseRepository) GetDiscussions(ctx context.Context, lessonID uuid.UUID) ([]Discussion, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, u.first_name || ' ' || u.last_name, u.avatar, u.role, d.content, d.upvotes, d.created_at
		FROM discussions d JOIN users u ON u.id=d.user_id WHERE d.lesson_id=$1 ORDER BY d.created_at DESC
	`, lessonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Discussion
	for rows.Next() {
		var d Discussion
		var created time.Time
		if err := rows.Scan(&d.ID, &d.UserName, &d.UserAvatar, &d.UserRole, &d.Content, &d.Upvotes, &created); err != nil {
			return nil, err
		}
		d.CreatedAt = utils.RelativeTime(created)
		d.Replies = r.replies(ctx, d.ID)
		out = append(out, d)
	}
	if out == nil {
		out = []Discussion{}
	}
	return out, nil
}

func (r *postgresCourseRepository) replies(ctx context.Context, discussionID uuid.UUID) []DiscussionReply {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, u.first_name || ' ' || u.last_name, u.avatar, u.role, r.content, r.created_at
		FROM discussion_replies r JOIN users u ON u.id=r.user_id WHERE r.discussion_id=$1 ORDER BY r.created_at
	`, discussionID)
	if err != nil {
		return []DiscussionReply{}
	}
	defer rows.Close()
	var out []DiscussionReply
	for rows.Next() {
		var rp DiscussionReply
		var created time.Time
		if err := rows.Scan(&rp.ID, &rp.UserName, &rp.UserAvatar, &rp.UserRole, &rp.Content, &created); err != nil {
			continue
		}
		rp.CreatedAt = utils.RelativeTime(created)
		out = append(out, rp)
	}
	if out == nil {
		return []DiscussionReply{}
	}
	return out
}

func (r *postgresCourseRepository) AddDiscussion(ctx context.Context, lessonID, courseID, userID uuid.UUID, content string) (*Discussion, error) {
	id := uuid.New()
	var created time.Time
	err := r.db.QueryRowContext(ctx, `INSERT INTO discussions (id, lesson_id, course_id, user_id, content) VALUES ($1,$2,$3,$4,$5) RETURNING created_at`,
		id, lessonID, courseID, userID, content).Scan(&created)
	if err != nil {
		return nil, err
	}
	list, _ := r.GetDiscussions(ctx, lessonID)
	for i := range list {
		if list[i].ID == id {
			return &list[i], nil
		}
	}
	return &Discussion{ID: id, Content: content, CreatedAt: utils.RelativeTime(created), Replies: []DiscussionReply{}}, nil
}

func (r *postgresCourseRepository) AddReply(ctx context.Context, discussionID, userID uuid.UUID, content string) (*DiscussionReply, error) {
	id := uuid.New()
	var created time.Time
	err := r.db.QueryRowContext(ctx, `INSERT INTO discussion_replies (id, discussion_id, user_id, content) VALUES ($1,$2,$3,$4) RETURNING created_at`,
		id, discussionID, userID, content).Scan(&created)
	if err != nil {
		return nil, err
	}
	var rp DiscussionReply
	_ = r.db.QueryRowContext(ctx, `SELECT u.first_name || ' ' || u.last_name, u.avatar, u.role FROM users u WHERE u.id=$1`, userID).
		Scan(&rp.UserName, &rp.UserAvatar, &rp.UserRole)
	rp.ID = id
	rp.Content = content
	rp.CreatedAt = utils.RelativeTime(created)
	return &rp, nil
}

func (r *postgresCourseRepository) ToggleUpvote(ctx context.Context, discussionID, userID uuid.UUID) (int, error) {
	var exists bool
	_ = r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM discussion_votes WHERE discussion_id=$1 AND user_id=$2)`, discussionID, userID).Scan(&exists)
	if exists {
		_, _ = r.db.ExecContext(ctx, `DELETE FROM discussion_votes WHERE discussion_id=$1 AND user_id=$2`, discussionID, userID)
		_, _ = r.db.ExecContext(ctx, `UPDATE discussions SET upvotes = GREATEST(upvotes-1,0) WHERE id=$1`, discussionID)
	} else {
		_, _ = r.db.ExecContext(ctx, `INSERT INTO discussion_votes (discussion_id, user_id) VALUES ($1,$2)`, discussionID, userID)
		_, _ = r.db.ExecContext(ctx, `UPDATE discussions SET upvotes = upvotes+1 WHERE id=$1`, discussionID)
	}
	var n int
	_ = r.db.QueryRowContext(ctx, `SELECT upvotes FROM discussions WHERE id=$1`, discussionID).Scan(&n)
	return n, nil
}

func (r *postgresCourseRepository) IsEnrolled(ctx context.Context, userID, courseID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM enrollments WHERE user_id=$1 AND course_id=$2)`, userID, courseID).Scan(&exists)
	return exists, err
}

func (r *postgresCourseRepository) CompletedLessonIDs(ctx context.Context, userID, courseID uuid.UUID) (map[uuid.UUID]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT lesson_id FROM lesson_progress WHERE user_id=$1 AND course_id=$2 AND completed=true`, userID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[uuid.UUID]bool{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		m[id] = true
	}
	return m, nil
}

func (r *postgresCourseRepository) GetEnrollmentProgress(ctx context.Context, userID, courseID uuid.UUID) float64 {
	var p float64
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(progress,0) FROM enrollments WHERE user_id=$1 AND course_id=$2`, userID, courseID).Scan(&p)
	return p
}

func (r *postgresCourseRepository) CourseIDForModule(ctx context.Context, moduleID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT course_id FROM modules WHERE id=$1`, moduleID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, appErrors.ErrModuleNotFound
	}
	return id, err
}

func (r *postgresCourseRepository) CourseIDForLesson(ctx context.Context, lessonID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	var courseID, moduleID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT m.course_id, l.module_id FROM lessons l JOIN modules m ON m.id=l.module_id WHERE l.id=$1`, lessonID).Scan(&courseID, &moduleID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, uuid.Nil, appErrors.ErrLessonNotFound
	}
	return courseID, moduleID, err
}

func (r *postgresCourseRepository) OwnerIDForCourse(ctx context.Context, courseID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT instructor_id FROM courses WHERE id=$1 AND deleted_at IS NULL`, courseID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, appErrors.ErrCourseNotFound
	}
	return id, err
}

func (r *postgresCourseRepository) ListInstructorCourses(ctx context.Context, instructorID uuid.UUID, isAdmin bool) ([]Course, error) {
	f := ListFilter{Page: 1, Limit: 500, IncludeUnpublished: true, ViewerRole: "instructor"}
	if !isAdmin {
		f.InstructorID = &instructorID
	}
	list, _, err := r.ListCourses(ctx, f)
	return list, err
}

func (r *postgresCourseRepository) GetEnrolledCourses(ctx context.Context, userID uuid.UUID) ([]EnrolledCourse, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT e.course_id, e.enrolled_at, e.progress, e.payment_status,
		       (SELECT COUNT(*) FROM modules m WHERE m.course_id=e.course_id),
		       (SELECT COUNT(DISTINCT m.id) FROM modules m JOIN lessons l ON l.module_id=m.id
		         WHERE m.course_id=e.course_id AND NOT EXISTS (
		           SELECT 1 FROM lessons l2 WHERE l2.module_id=m.id AND l2.is_published=true
		           AND NOT EXISTS (SELECT 1 FROM lesson_progress lp WHERE lp.user_id=e.user_id AND lp.lesson_id=l2.id AND lp.completed=true)
		         )),
		       (SELECT COUNT(*) FROM lessons l JOIN modules m ON m.id=l.module_id WHERE m.course_id=e.course_id AND l.quiz_set_id IS NOT NULL),
		       (SELECT COUNT(*) FROM quiz_attempts qa WHERE qa.user_id=e.user_id AND qa.course_id=e.course_id AND qa.passed=true),
		       COALESCE((SELECT AVG(qa.score) FROM quiz_attempts qa WHERE qa.user_id=e.user_id AND qa.course_id=e.course_id),0)
		FROM enrollments e WHERE e.user_id=$1 ORDER BY e.enrolled_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EnrolledCourse
	for rows.Next() {
		var ec EnrolledCourse
		var enrolledAt time.Time
		var courseID uuid.UUID
		if err := rows.Scan(&courseID, &enrolledAt, &ec.Progress, &ec.PaymentStatus, &ec.TotalModules, &ec.CompletedModules, &ec.TotalQuizzes, &ec.CompletedQuizzes, &ec.QuizScore); err != nil {
			return nil, err
		}
		c, err := r.GetByID(ctx, courseID)
		if err != nil {
			continue
		}
		ec.Course = *c
		ec.Progress = c.Progress
		if ec.Progress == 0 {
			ec.Progress = r.GetEnrollmentProgress(ctx, userID, courseID)
			ec.Course.Progress = ec.Progress
		}
		ec.EnrolledDate = enrolledAt.Format("2006-01-02")
		ec.OtherScore = 10
		if ec.Progress >= 100 {
			ec.OtherScore = 10
		} else {
			ec.OtherScore = 0
		}
		ec.TotalScore = ec.QuizScore + ec.OtherScore
		out = append(out, ec)
	}
	if out == nil {
		out = []EnrolledCourse{}
	}
	return out, nil
}
