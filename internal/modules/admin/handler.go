package admin

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/core/response"
	"nexura-backend/internal/modules/auth"
	"nexura-backend/internal/modules/course"
	"nexura-backend/internal/modules/payment"
	"nexura-backend/pkg/utils"
)

type AdminOverviewStats struct {
	TotalRevenue         float64            `json:"totalRevenue"`
	AdminNetCommission   float64            `json:"adminNetCommission"`
	InstructorPayouts    float64            `json:"instructorPayouts"`
	DummyWalletBalance   float64            `json:"dummyWalletBalance"`
	IsDummyPayments      bool               `json:"isDummyPayments"`
	TotalStudents        int                `json:"totalStudents"`
	TotalInstructors     int                `json:"totalInstructors"`
	TotalCourses         int                `json:"totalCourses"`
	ActiveEnrollments    int                `json:"activeEnrollments"`
	GrowthRate           map[string]float64 `json:"growthRate"`
	MonthlyGrowth        []MonthlyGrowth    `json:"monthlyGrowth,omitempty"`
	CategoryStats        []CategoryStat     `json:"categoryStats,omitempty"`
	RecentTransactions   []payment.PlatformTransaction `json:"recentTransactions,omitempty"`
}

type MonthlyGrowth struct {
	Month              string  `json:"month"`
	GMV                float64 `json:"gmv"`
	AdminRevenue       float64 `json:"adminRevenue"`
	InstructorEarnings float64 `json:"instructorEarnings"`
	Students           int     `json:"students"`
	Enrollments        int     `json:"enrollments"`
}

type CategoryStat struct {
	Name    string  `json:"name"`
	Count   int     `json:"count"`
	Revenue float64 `json:"revenue"`
	Color   string  `json:"color"`
}

type AdminHandler struct {
	db         *sql.DB
	userRepo   auth.UserRepository
	courseRepo course.CourseRepository
	payRepo    payment.Repository
}

func NewAdminHandler(db *sql.DB, userRepo auth.UserRepository, courseRepo course.CourseRepository, payRepo payment.Repository) *AdminHandler {
	return &AdminHandler{db: db, userRepo: userRepo, courseRepo: courseRepo, payRepo: payRepo}
}

func (h *AdminHandler) GetOverviewStats(c *gin.Context) {
	stats, err := h.overview(c)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", stats)
}

func (h *AdminHandler) overview(c *gin.Context) (*AdminOverviewStats, error) {
	ctx := c.Request.Context()
	s := &AdminOverviewStats{IsDummyPayments: true, GrowthRate: map[string]float64{"revenue": 0, "students": 0, "courses": 0}}
	_ = h.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(price),0), COALESCE(SUM(admin_commission_amount),0), COALESCE(SUM(instructor_earnings),0) FROM transactions WHERE status='completed'`).
		Scan(&s.TotalRevenue, &s.AdminNetCommission, &s.InstructorPayouts)
	_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role='student' AND deleted_at IS NULL`).Scan(&s.TotalStudents)
	_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role='instructor' AND deleted_at IS NULL`).Scan(&s.TotalInstructors)
	_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM courses WHERE deleted_at IS NULL`).Scan(&s.TotalCourses)
	_ = h.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM enrollments`).Scan(&s.ActiveEnrollments)
	if h.payRepo != nil {
		if w, err := h.payRepo.GetWallet(ctx, "admin", uuid.Nil); err == nil {
			s.DummyWalletBalance = w.Balance
		}
	}
	s.MonthlyGrowth = h.monthly(ctx)
	s.CategoryStats = h.categories(ctx)
	s.RecentTransactions = h.txns(ctx, "", "", 5, 0)
	if len(s.MonthlyGrowth) >= 2 {
		prev := s.MonthlyGrowth[len(s.MonthlyGrowth)-2]
		cur := s.MonthlyGrowth[len(s.MonthlyGrowth)-1]
		if prev.GMV > 0 {
			s.GrowthRate["revenue"] = utils.RoundMoney((cur.GMV - prev.GMV) / prev.GMV * 100)
		}
		if prev.Students > 0 {
			s.GrowthRate["students"] = utils.RoundMoney(float64(cur.Students-prev.Students) / float64(prev.Students) * 100)
		}
	}
	return s, nil
}

func (h *AdminHandler) monthly(ctx context.Context) []MonthlyGrowth {
	rows, err := h.db.QueryContext(ctx, `
		SELECT to_char(created_at, 'Mon'), COALESCE(SUM(price),0), COALESCE(SUM(admin_commission_amount),0), COALESCE(SUM(instructor_earnings),0),
		       COUNT(DISTINCT student_id), COUNT(*)
		FROM transactions WHERE status='completed' AND created_at > NOW() - INTERVAL '12 months'
		GROUP BY to_char(created_at, 'Mon'), date_trunc('month', created_at)
		ORDER BY date_trunc('month', created_at)
	`)
	if err != nil {
		return []MonthlyGrowth{}
	}
	defer rows.Close()
	out := []MonthlyGrowth{}
	for rows.Next() {
		var m MonthlyGrowth
		if err := rows.Scan(&m.Month, &m.GMV, &m.AdminRevenue, &m.InstructorEarnings, &m.Students, &m.Enrollments); err == nil {
			out = append(out, m)
		}
	}
	return out
}

func (h *AdminHandler) categories(ctx context.Context) []CategoryStat {
	colors := []string{"#0ea5e9", "#8b5cf6", "#f59e0b", "#10b981", "#ef4444", "#6366f1", "#ec4899", "#14b8a6"}
	rows, err := h.db.QueryContext(ctx, `
		SELECT COALESCE(cat.title,'Uncategorized'), COUNT(c.id), COALESCE(SUM(t.price),0)
		FROM courses c
		LEFT JOIN categories cat ON cat.id=c.category_id
		LEFT JOIN transactions t ON t.course_id=c.id AND t.status='completed'
		WHERE c.deleted_at IS NULL
		GROUP BY cat.title
	`)
	if err != nil {
		return []CategoryStat{}
	}
	defer rows.Close()
	out := []CategoryStat{}
	i := 0
	for rows.Next() {
		var s CategoryStat
		if err := rows.Scan(&s.Name, &s.Count, &s.Revenue); err == nil {
			s.Color = colors[i%len(colors)]
			out = append(out, s)
			i++
		}
	}
	return out
}

func (h *AdminHandler) txns(ctx context.Context, search, creatorType string, limit, offset int) []payment.PlatformTransaction {
	q := `
		SELECT COALESCE(p.public_txn_id, t.id::text), t.course_id, t.course_title, t.creator_type, t.instructor_name,
		       t.student_name, t.student_email, t.price, t.admin_commission_rate, t.admin_commission_amount, t.instructor_earnings,
		       t.created_at, t.status, t.payment_method
		FROM transactions t
		LEFT JOIN payments p ON p.id=t.payment_id
		WHERE t.status='completed'
	`
	args := []any{}
	i := 1
	if creatorType != "" {
		q += ` AND t.creator_type=$` + strconv.Itoa(i)
		args = append(args, creatorType)
		i++
	}
	if search != "" {
		q += ` AND (COALESCE(p.public_txn_id,'') ILIKE $` + strconv.Itoa(i) + ` OR t.student_name ILIKE $` + strconv.Itoa(i) + ` OR t.course_title ILIKE $` + strconv.Itoa(i) + `)`
		args = append(args, "%"+search+"%")
		i++
	}
	q += ` ORDER BY t.created_at DESC LIMIT $` + strconv.Itoa(i) + ` OFFSET $` + strconv.Itoa(i+1)
	args = append(args, limit, offset)
	rows, err := h.db.QueryContext(ctx, q, args...)
	if err != nil {
		return []payment.PlatformTransaction{}
	}
	defer rows.Close()
	out := []payment.PlatformTransaction{}
	for rows.Next() {
		var t payment.PlatformTransaction
		var created time.Time
		var courseID uuid.UUID
		if err := rows.Scan(&t.ID, &courseID, &t.CourseTitle, &t.CreatorType, &t.InstructorName, &t.StudentName, &t.StudentEmail, &t.Price, &t.AdminCommissionRate, &t.AdminCommissionAmount, &t.InstructorEarnings, &created, &t.Status, &t.PaymentMethod); err == nil {
			t.CourseID = courseID.String()
			t.Date = created.Format("2006-01-02")
			t.PaymentMethod = "Dummy"
			out = append(out, t)
		}
	}
	return out
}

func (h *AdminHandler) GetRevenueAnalytics(c *gin.Context) {
	ctx := c.Request.Context()
	search := c.Query("search")
	creatorType := c.Query("creatorType")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit < 1 {
		limit = 50
	}
	var gmv, adminCut, instr float64
	_ = h.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(price),0), COALESCE(SUM(admin_commission_amount),0), COALESCE(SUM(instructor_earnings),0) FROM transactions WHERE status='completed'`).
		Scan(&gmv, &adminCut, &instr)
	var instructorSales, admin5, adminDirect float64
	_ = h.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(price),0), COALESCE(SUM(admin_commission_amount),0) FROM transactions WHERE status='completed' AND creator_type='instructor'`).Scan(&instructorSales, &admin5)
	_ = h.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(price),0) FROM transactions WHERE status='completed' AND creator_type='admin'`).Scan(&adminDirect)
	walletBal := 0.0
	if h.payRepo != nil {
		if w, err := h.payRepo.GetWallet(ctx, "admin", uuid.Nil); err == nil {
			walletBal = w.Balance
		}
	}
	txns := h.txns(ctx, search, creatorType, limit, utils.Offset(page, limit))
	response.Success(c, http.StatusOK, "OK", gin.H{
		"totalGmv": gmv, "adminNetCommission": adminCut, "instructorPayouts": instr,
		"dummyWalletBalance": walletBal, "isDummyPayments": true,
		"instructorCourseSales": instructorSales, "admin5PercentCut": admin5, "adminDirectCourseSales": adminDirect,
		"monthlyGrowth": h.monthly(ctx), "transactions": txns,
	})
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	rows, total, err := h.userRepo.ListUsers(c.Request.Context(), c.Query("role"), c.Query("search"), page, limit)
	if err != nil {
		response.Internal(c, err)
		return
	}
	out := []gin.H{}
	for _, u := range rows {
		out = append(out, gin.H{
			"id": u.ID, "firstName": u.FirstName, "lastName": u.LastName, "email": u.Email, "role": u.Role,
			"avatar": u.Avatar, "status": u.Status, "joinDate": u.JoinDate.Format("2006-01-02"),
			"enrolledCoursesCount": u.EnrolledCoursesCount, "totalSpent": u.TotalSpent,
			"createdCoursesCount": u.CreatedCoursesCount, "totalStudentsCount": u.TotalStudentsCount,
			"totalEarnings": u.TotalEarnings, "adminCommissionGenerated": u.AdminCommissionGenerated,
			"phone": u.Phone, "bio": u.Bio,
		})
	}
	response.SuccessWithMeta(c, http.StatusOK, "OK", out, utils.PageMeta(page, limit, total))
}

func (h *AdminHandler) GetUserByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	u, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "OK", u.Public())
}

func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	var req struct {
		Role auth.UserRole `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	target, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	if target.Role == auth.RoleAdmin && req.Role != auth.RoleAdmin {
		n, _ := h.userRepo.CountAdmins(c.Request.Context())
		if n <= 1 {
			response.Unprocessable(c, appErrors.ErrLastAdmin.Error())
			return
		}
	}
	if err := h.userRepo.UpdateRole(c.Request.Context(), id, req.Role); err != nil {
		response.Internal(c, err)
		return
	}
	if req.Role == auth.RoleInstructor {
		_ = h.userRepo.EnsureWallet(c.Request.Context(), "instructor", id)
	}
	response.Success(c, http.StatusOK, "User role updated successfully", gin.H{"userId": id, "role": req.Role})
}

func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	var req struct {
		Status auth.UserStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.userRepo.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "User status updated successfully", gin.H{"userId": id, "status": req.Status})
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	self, _ := middleware.CurrentUserID(c)
	if self == id {
		response.Unprocessable(c, appErrors.ErrCannotDeleteSelf.Error())
		return
	}
	target, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	if target.Role == auth.RoleAdmin {
		n, _ := h.userRepo.CountAdmins(c.Request.Context())
		if n <= 1 {
			response.Unprocessable(c, appErrors.ErrLastAdmin.Error())
			return
		}
	}
	if err := h.userRepo.SoftDelete(c.Request.Context(), id); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "User deleted successfully", gin.H{})
}

func (h *AdminHandler) ApproveCourse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.courseRepo.SetPublished(c.Request.Context(), id, true); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Course approved and published successfully", gin.H{"courseId": id, "isPublished": true})
}

func (h *AdminHandler) InstructorAnalytics(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var earnings float64
	var students, courses int
	_ = h.db.QueryRowContext(c, `SELECT COALESCE(SUM(instructor_earnings),0) FROM transactions WHERE instructor_id=$1 AND status='completed'`, userID).Scan(&earnings)
	_ = h.db.QueryRowContext(c, `SELECT COUNT(DISTINCT e.user_id) FROM enrollments e JOIN courses c ON c.id=e.course_id WHERE c.instructor_id=$1`, userID).Scan(&students)
	_ = h.db.QueryRowContext(c, `SELECT COUNT(*) FROM courses WHERE instructor_id=$1 AND deleted_at IS NULL`, userID).Scan(&courses)
	response.Success(c, http.StatusOK, "OK", gin.H{"instructorEarnings": earnings, "totalStudents": students, "totalCourses": courses})
}

func (h *AdminHandler) StudentProgress(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	role := middleware.CurrentRole(c)
	q := `
		SELECT u.first_name || ' ' || u.last_name, c.title, e.progress, e.enrolled_at
		FROM enrollments e JOIN users u ON u.id=e.user_id JOIN courses c ON c.id=e.course_id
	`
	args := []any{}
	if role != "admin" {
		q += ` WHERE c.instructor_id=$1`
		args = append(args, userID)
	}
	q += ` ORDER BY e.enrolled_at DESC LIMIT 100`
	rows, err := h.db.QueryContext(c, q, args...)
	if err != nil {
		response.Internal(c, err)
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var name, title string
		var progress float64
		var at time.Time
		if err := rows.Scan(&name, &title, &progress, &at); err == nil {
			list = append(list, gin.H{"studentName": name, "courseTitle": title, "progress": progress, "enrolledDate": at.Format("2006-01-02")})
		}
	}
	response.Success(c, http.StatusOK, "OK", list)
}
