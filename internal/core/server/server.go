package server

import (
	"github.com/gin-gonic/gin"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/modules/admin"
	"nexura-backend/internal/modules/auth"
	"nexura-backend/internal/modules/certificate"
	"nexura-backend/internal/modules/chat"
	"nexura-backend/internal/modules/course"
	"nexura-backend/internal/modules/live"
	"nexura-backend/internal/modules/notification"
	"nexura-backend/internal/modules/payment"
	"nexura-backend/internal/modules/quiz"
	"nexura-backend/internal/modules/instructor"
	"nexura-backend/internal/modules/upload"
	"nexura-backend/pkg/jwt"
)

type ServerConfig struct {
	Engine            *gin.Engine
	JWTService        *jwt.JWTService
	ClientOrigin      string
	AuthHandler       *auth.AuthHandler
	CourseHandler     *course.CourseHandler
	InstructorHandler *instructor.InstructorHandler
	AdminHandler      *admin.AdminHandler
	ChatHandler       *chat.ChatHandler
	WSHandler         *chat.WSHandler
	UploadHandler     *upload.UploadHandler
	PaymentHandler    *payment.Handler
	QuizHandler       *quiz.Handler
	LiveHandler       *live.Handler
	NotifHandler      *notification.Handler
	CertHandler       *certificate.Handler
}

func SetupServer(cfg ServerConfig) *gin.Engine {
	origin := cfg.ClientOrigin
	if origin == "" {
		origin = "*"
	}
	cfg.Engine.Use(middleware.CORSWithOrigin(origin))
	cfg.Engine.Static("/uploads", "./uploads")

	cfg.Engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "Nexura Hub API Server"})
	})
	if cfg.WSHandler != nil {
		cfg.Engine.GET("/ws", cfg.WSHandler.ServeWS)
	}

	jwtMw := middleware.JWTAuthMiddleware(cfg.JWTService)
	optJwt := middleware.OptionalJWTMiddleware(cfg.JWTService)
	adminOnly := []gin.HandlerFunc{jwtMw, middleware.RequireRole("admin")}
	studio := []gin.HandlerFunc{jwtMw, middleware.RequireRole("instructor", "admin")}
	chatMw := []gin.HandlerFunc{jwtMw, middleware.BlockAdminChatMiddleware()}

	api := cfg.Engine.Group("/api/v1")
	{
		authG := api.Group("/auth")
		{
			authG.POST("/register", cfg.AuthHandler.Register)
			authG.POST("/login", cfg.AuthHandler.Login)
			authG.POST("/logout", optJwt, cfg.AuthHandler.Logout)
			authG.POST("/refresh-token", cfg.AuthHandler.RefreshToken)
			authG.POST("/forgot-password", cfg.AuthHandler.ForgotPassword)
			authG.POST("/reset-password", cfg.AuthHandler.ResetPassword)
			authG.POST("/verify-email", cfg.AuthHandler.VerifyEmail)
			authG.GET("/me", jwtMw, cfg.AuthHandler.GetMe)
			authG.PUT("/profile", jwtMw, cfg.AuthHandler.UpdateProfile)
			authG.POST("/change-password", jwtMw, cfg.AuthHandler.ChangePassword)
		}

		api.GET("/categories", cfg.CourseHandler.GetCategories)
		api.GET("/categories/:id", cfg.CourseHandler.GetCategory)
		api.POST("/categories", append(adminOnly, cfg.CourseHandler.CreateCategory)...)
		api.PUT("/categories/:id", append(adminOnly, cfg.CourseHandler.UpdateCategory)...)
		api.DELETE("/categories/:id", append(adminOnly, cfg.CourseHandler.DeleteCategory)...)

		api.GET("/courses", optJwt, cfg.CourseHandler.ListCourses)
		api.GET("/courses/categories", cfg.CourseHandler.GetCategories)
		api.GET("/courses/enrolled", jwtMw, cfg.CourseHandler.GetEnrolledCourses)
		api.GET("/courses/:id", optJwt, cfg.CourseHandler.GetCourseByIDOrSlug)
		api.GET("/courses/:id/reviews", cfg.CourseHandler.GetReviews)
		api.GET("/courses/:id/modules", optJwt, cfg.CourseHandler.ListModules)
		api.GET("/courses/:courseId/modules", optJwt, cfg.CourseHandler.ListModules)
		api.GET("/courses/:courseId/enrollments", append(studio, cfg.PaymentHandler.CourseEnrollments)...)
		api.POST("/courses", jwtMw, middleware.RequireRole("instructor", "admin"), cfg.CourseHandler.CreateCourse)
		api.PUT("/courses/:id", jwtMw, cfg.CourseHandler.UpdateCourse)
		api.DELETE("/courses/:id", jwtMw, cfg.CourseHandler.DeleteCourse)
		api.PATCH("/courses/:id/publish", jwtMw, cfg.CourseHandler.TogglePublish)
		api.PATCH("/courses/:id/unpublish", jwtMw, cfg.CourseHandler.Unpublish)
		api.POST("/courses/:id/modules", jwtMw, cfg.CourseHandler.AddModule)
		api.PUT("/courses/:courseId/modules/reorder", jwtMw, cfg.CourseHandler.ReorderModules)
		api.POST("/courses/:id/reviews", jwtMw, cfg.CourseHandler.AddReview)
		api.POST("/courses/:id/enroll", jwtMw, cfg.PaymentHandler.EnrollByCourseParam)

		api.GET("/modules/:id", jwtMw, cfg.CourseHandler.GetModule)
		api.GET("/modules/:moduleId", jwtMw, cfg.CourseHandler.GetModule)
		api.PUT("/modules/:id", jwtMw, cfg.CourseHandler.UpdateModule)
		api.DELETE("/modules/:id", jwtMw, cfg.CourseHandler.DeleteModule)
		api.POST("/modules/:id/lessons", jwtMw, cfg.CourseHandler.AddLesson)
		api.PUT("/modules/:moduleId/lessons/reorder", jwtMw, cfg.CourseHandler.ReorderLessons)

		api.GET("/lessons/:lessonId", optJwt, cfg.CourseHandler.GetLesson)
		api.PUT("/lessons/:id", jwtMw, cfg.CourseHandler.UpdateLesson)
		api.DELETE("/lessons/:id", jwtMw, cfg.CourseHandler.DeleteLesson)
		api.POST("/lessons/:id/complete", jwtMw, cfg.PaymentHandler.CompleteLesson)
		api.PATCH("/lessons/:lessonId/complete", jwtMw, cfg.PaymentHandler.CompleteLesson)
		api.POST("/lessons/:lessonId/resources", jwtMw, cfg.CourseHandler.AddResource)
		api.DELETE("/lessons/:lessonId/resources/:resourceId", jwtMw, cfg.CourseHandler.DeleteResource)
		api.GET("/lessons/:lessonId/notes", jwtMw, cfg.CourseHandler.GetNotes)
		api.POST("/lessons/:lessonId/notes", jwtMw, cfg.CourseHandler.AddNote)
		api.DELETE("/lessons/:lessonId/notes/:noteId", jwtMw, cfg.CourseHandler.DeleteNote)
		api.GET("/lessons/:lessonId/discussions", jwtMw, cfg.CourseHandler.GetDiscussions)
		api.POST("/lessons/:lessonId/discussions", jwtMw, cfg.CourseHandler.AddDiscussion)
		api.POST("/discussions/:id/replies", jwtMw, cfg.CourseHandler.AddReply)
		api.POST("/discussions/:id/upvote", jwtMw, cfg.CourseHandler.ToggleUpvote)
		api.PUT("/reviews/:id", jwtMw, cfg.CourseHandler.UpdateReview)
		api.DELETE("/reviews/:id", jwtMw, cfg.CourseHandler.DeleteReview)

		dash := api.Group("/dashboard", studio...)
		{
			dash.GET("/stats", cfg.InstructorHandler.GetDashboardStats)
			dash.GET("/courses", cfg.InstructorHandler.GetDashboardCourses)
			dash.GET("/lives", cfg.InstructorHandler.GetDashboardLives)
			dash.POST("/lives", cfg.LiveHandler.Create)
			dash.GET("/quiz-sets", cfg.InstructorHandler.GetDashboardQuizSets)
			dash.GET("/enrollments", cfg.InstructorHandler.GetDashboardEnrollments)
			dash.GET("/wallet", cfg.PaymentHandler.InstructorWallet)
		}
		instr := api.Group("/instructor", studio...)
		{
			instr.GET("/dashboard/stats", cfg.InstructorHandler.GetDashboardStats)
			instr.GET("/dashboard/courses", cfg.InstructorHandler.GetDashboardCourses)
			instr.GET("/dashboard/lives", cfg.InstructorHandler.GetDashboardLives)
			instr.GET("/dashboard/quiz-sets", cfg.InstructorHandler.GetDashboardQuizSets)
			instr.GET("/dashboard/enrollments", cfg.InstructorHandler.GetDashboardEnrollments)
			instr.POST("/courses", cfg.CourseHandler.CreateCourse)
		}

		api.GET("/lives/:id", jwtMw, cfg.LiveHandler.Get)
		api.POST("/lives", append(studio, cfg.LiveHandler.Create)...)
		api.PUT("/lives/:id", append(studio, cfg.LiveHandler.Update)...)
		api.DELETE("/lives/:id", append(studio, cfg.LiveHandler.Delete)...)

		api.GET("/quizzes", append(studio, cfg.QuizHandler.List)...)
		api.GET("/quizzes/:quizId", jwtMw, cfg.QuizHandler.Get)
		api.POST("/quizzes", append(studio, cfg.QuizHandler.Create)...)
		api.PUT("/quizzes/:quizId", append(studio, cfg.QuizHandler.Update)...)
		api.DELETE("/quizzes/:quizId", append(studio, cfg.QuizHandler.Delete)...)
		api.DELETE("/quizzes/:id", append(studio, cfg.QuizHandler.Delete)...)
		api.POST("/quizzes/:quizId/questions", append(studio, cfg.QuizHandler.AddQuestion)...)
		api.DELETE("/quizzes/:quizId/questions/:qId", append(studio, cfg.QuizHandler.DeleteQuestion)...)
		api.POST("/quizzes/:quizId/submit", jwtMw, cfg.QuizHandler.Submit)

		api.GET("/analytics/overview", append(adminOnly, cfg.AdminHandler.GetOverviewStats)...)
		api.GET("/analytics/revenue", append(adminOnly, cfg.AdminHandler.GetRevenueAnalytics)...)
		api.GET("/analytics/instructor", jwtMw, middleware.RequireRole("instructor", "admin"), cfg.AdminHandler.InstructorAnalytics)
		api.GET("/analytics/student-progress", jwtMw, middleware.RequireRole("instructor", "admin"), cfg.AdminHandler.StudentProgress)

		users := api.Group("/users", adminOnly...)
		{
			users.GET("", cfg.AdminHandler.ListUsers)
			users.GET("/:id", cfg.AdminHandler.GetUserByID)
			users.PATCH("/:id/role", cfg.AdminHandler.UpdateUserRole)
			users.PATCH("/:id/status", cfg.AdminHandler.UpdateUserStatus)
			users.DELETE("/:id", cfg.AdminHandler.DeleteUser)
		}

		adminG := api.Group("/admin", adminOnly...)
		{
			adminG.GET("/overview", cfg.AdminHandler.GetOverviewStats)
			adminG.GET("/wallet", cfg.PaymentHandler.AdminWallet)
			adminG.PATCH("/courses/:id/approve", cfg.AdminHandler.ApproveCourse)
			adminG.PATCH("/users/:id/status", cfg.AdminHandler.UpdateUserStatus)
		}

		api.GET("/coupons", append(adminOnly, cfg.PaymentHandler.ListCoupons)...)
		api.POST("/coupons", append(adminOnly, cfg.PaymentHandler.CreateCoupon)...)
		api.PATCH("/coupons/:id", append(adminOnly, cfg.PaymentHandler.PatchCoupon)...)
		api.DELETE("/coupons/:id", append(adminOnly, cfg.PaymentHandler.DeleteCoupon)...)
		api.POST("/coupons/validate", jwtMw, cfg.PaymentHandler.ValidateCoupon)

		api.POST("/payments/dummy", jwtMw, cfg.PaymentHandler.DummyPay)
		api.POST("/payments/init", jwtMw, cfg.PaymentHandler.DummyPay)
		api.GET("/payments/:id", jwtMw, cfg.PaymentHandler.GetPayment)

		api.POST("/enrollments", jwtMw, cfg.PaymentHandler.FreeEnroll)
		api.GET("/enrollments/:id/status", jwtMw, cfg.PaymentHandler.EnrollmentStatus)
		api.GET("/enrollments/:courseId/status", jwtMw, cfg.PaymentHandler.EnrollmentStatus)
		api.GET("/user/enrolled-courses", jwtMw, cfg.CourseHandler.GetEnrolledCourses)

		api.GET("/notifications", jwtMw, cfg.NotifHandler.List)
		api.PATCH("/notifications/:id/read", jwtMw, cfg.NotifHandler.Read)
		api.POST("/notifications/read-all", jwtMw, cfg.NotifHandler.ReadAll)
		api.GET("/notifications/unread-count", jwtMw, cfg.NotifHandler.UnreadCount)

		api.POST("/certificates/:courseId", jwtMw, cfg.CertHandler.Issue)
		api.GET("/certificates/me/:courseId", jwtMw, cfg.CertHandler.Me)
		api.GET("/certificates/verify/:publicId", cfg.CertHandler.Verify)

		api.GET("/instructors/:id", cfg.AuthHandler.GetInstructorPublic)
		api.GET("/instructors/:id/courses", optJwt, cfg.CourseHandler.ListCourses)

		up := api.Group("/upload", jwtMw)
		{
			up.POST("/image", cfg.UploadHandler.UploadImage)
			up.POST("/avatar", cfg.UploadHandler.UploadAvatar)
			up.POST("/video", cfg.UploadHandler.UploadVideo)
			up.POST("/pdf", cfg.UploadHandler.UploadPDF)
			up.POST("/file", cfg.UploadHandler.UploadFile)
			up.POST("/delete", cfg.UploadHandler.DeleteFile)
			up.DELETE("/delete", cfg.UploadHandler.DeleteFile)
		}

		chatG := api.Group("/chat", chatMw...)
		{
			chatG.GET("/conversations", cfg.ChatHandler.GetConversations)
			chatG.GET("/conversations/:id", cfg.ChatHandler.GetConversation)
			chatG.GET("/conversations/:id/messages", cfg.ChatHandler.GetMessages)
			chatG.POST("/conversations/direct", cfg.ChatHandler.CreateDirect)
			chatG.POST("/conversations/:id/read", cfg.ChatHandler.MarkRead)
			chatG.POST("/groups", middleware.RequireRole("instructor", "admin"), cfg.ChatHandler.CreateGroup)
			chatG.POST("/groups/:id/join", cfg.ChatHandler.JoinGroup)
			chatG.POST("/messages", cfg.ChatHandler.SendMessage)
			chatG.DELETE("/messages/:id", cfg.ChatHandler.DeleteMessage)
			chatG.POST("/messages/:id/reactions", cfg.ChatHandler.SendReaction)
			chatG.POST("/upload-image", cfg.UploadHandler.UploadImage)
		}
		legacy := api.Group("/conversations", chatMw...)
		{
			legacy.GET("", cfg.ChatHandler.GetConversations)
			legacy.GET("/:id/messages", cfg.ChatHandler.GetMessages)
			legacy.POST("/messages", cfg.ChatHandler.SendMessage)
			legacy.POST("/reactions", cfg.ChatHandler.SendReaction)
		}
	}

	return cfg.Engine
}
