// internal/core/server/server.go
package server

import (
	"github.com/gin-gonic/gin"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/modules/admin"
	"nexura-backend/internal/modules/auth"
	"nexura-backend/internal/modules/chat"
	"nexura-backend/internal/modules/course"
	"nexura-backend/internal/modules/enrollment"
	"nexura-backend/internal/modules/instructor"
	"nexura-backend/internal/modules/upload"
	"nexura-backend/pkg/jwt"
)

type ServerConfig struct {
	Engine            *gin.Engine
	JWTService        *jwt.JWTService
	AuthHandler       *auth.AuthHandler
	CourseHandler     *course.CourseHandler
	EnrollmentHandler *enrollment.EnrollmentHandler
	InstructorHandler *instructor.InstructorHandler
	AdminHandler      *admin.AdminHandler
	ChatHandler       *chat.ChatHandler
	WSHandler         *chat.WSHandler
	UploadHandler     *upload.UploadHandler
}

func SetupServer(cfg ServerConfig) *gin.Engine {
	cfg.Engine.Use(middleware.CORSMiddleware())

	cfg.Engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "Nexura Hub API Server"})
	})

	cfg.Engine.GET("/ws", cfg.WSHandler.ServeWS)

	api := cfg.Engine.Group("/api/v1")
	{
		// 1. Auth Group
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", cfg.AuthHandler.Register)
			authGroup.POST("/login", cfg.AuthHandler.Login)
			authGroup.POST("/logout", cfg.AuthHandler.Logout)
			authGroup.POST("/refresh-token", cfg.AuthHandler.RefreshToken)
			authGroup.POST("/forgot-password", cfg.AuthHandler.ForgotPassword)
			authGroup.POST("/reset-password", cfg.AuthHandler.ResetPassword)

			authGroup.GET("/me", middleware.JWTAuthMiddleware(cfg.JWTService), cfg.AuthHandler.GetMe)
			authGroup.PUT("/profile", middleware.JWTAuthMiddleware(cfg.JWTService), cfg.AuthHandler.UpdateProfile)
			authGroup.POST("/change-password", middleware.JWTAuthMiddleware(cfg.JWTService), cfg.AuthHandler.ChangePassword)
		}

		// 2. Instructor Dashboard Group (/dashboard & /instructor)
		dashGroup := api.Group("/dashboard")
		dashGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService), middleware.RequireRole("instructor", "admin"))
		{
			dashGroup.GET("/stats", cfg.InstructorHandler.GetDashboardStats)
			dashGroup.GET("/courses", cfg.InstructorHandler.GetDashboardCourses)
			dashGroup.GET("/lives", cfg.InstructorHandler.GetDashboardLives)
			dashGroup.GET("/quiz-sets", cfg.InstructorHandler.GetDashboardQuizSets)
			dashGroup.GET("/enrollments", cfg.InstructorHandler.GetDashboardEnrollments)
		}

		instructorGroup := api.Group("/instructor")
		instructorGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService), middleware.RequireRole("instructor", "admin"))
		{
			instructorGroup.GET("/dashboard/stats", cfg.InstructorHandler.GetDashboardStats)
			instructorGroup.GET("/dashboard/courses", cfg.InstructorHandler.GetDashboardCourses)
			instructorGroup.GET("/dashboard/lives", cfg.InstructorHandler.GetDashboardLives)
			instructorGroup.GET("/dashboard/quiz-sets", cfg.InstructorHandler.GetDashboardQuizSets)
			instructorGroup.GET("/dashboard/enrollments", cfg.InstructorHandler.GetDashboardEnrollments)
			instructorGroup.POST("/courses", cfg.CourseHandler.CreateCourse)
		}

		// 3. Admin & Analytics Group
		analyticsGroup := api.Group("/analytics")
		analyticsGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService), middleware.RequireRole("admin"))
		{
			analyticsGroup.GET("/overview", cfg.AdminHandler.GetOverviewStats)
			analyticsGroup.GET("/revenue", cfg.AdminHandler.GetRevenueAnalytics)
		}

		usersGroup := api.Group("/users")
		usersGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService), middleware.RequireRole("admin"))
		{
			usersGroup.GET("", cfg.AdminHandler.ListUsers)
			usersGroup.GET("/:id", cfg.AdminHandler.GetUserByID)
			usersGroup.PATCH("/:id/role", cfg.AdminHandler.UpdateUserRole)
			usersGroup.DELETE("/:id", cfg.AdminHandler.DeleteUser)
		}

		adminGroup := api.Group("/admin")
		adminGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService), middleware.RequireRole("admin"))
		{
			adminGroup.GET("/overview", cfg.AdminHandler.GetOverviewStats)
			adminGroup.PATCH("/courses/:id/approve", cfg.AdminHandler.ApproveCourse)
			adminGroup.PATCH("/users/:id/status", cfg.AdminHandler.UpdateUserStatus)
		}

		// 4. Course, Module & Lesson Management
		courseGroup := api.Group("/courses")
		{
			courseGroup.GET("", cfg.CourseHandler.ListCourses)
			courseGroup.GET("/categories", cfg.CourseHandler.GetCategories)
			courseGroup.GET("/:id", cfg.CourseHandler.GetCourseByIDOrSlug)

			// Authenticated course actions
			authedCourses := courseGroup.Group("")
			authedCourses.Use(middleware.JWTAuthMiddleware(cfg.JWTService))
			{
				authedCourses.POST("", cfg.CourseHandler.CreateCourse)
				authedCourses.PUT("/:id", cfg.CourseHandler.UpdateCourse)
				authedCourses.DELETE("/:id", cfg.CourseHandler.DeleteCourse)
				authedCourses.PATCH("/:id/publish", cfg.CourseHandler.TogglePublish)
				authedCourses.POST("/:id/modules", cfg.CourseHandler.AddModule)
			}
		}

		modulesGroup := api.Group("/modules")
		modulesGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService))
		{
			modulesGroup.PUT("/:id", cfg.CourseHandler.UpdateModule)
			modulesGroup.DELETE("/:id", cfg.CourseHandler.DeleteModule)
			modulesGroup.POST("/:id/lessons", cfg.CourseHandler.AddLesson)
		}

		lessonsGroup := api.Group("/lessons")
		lessonsGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService))
		{
			lessonsGroup.PUT("/:id", cfg.CourseHandler.UpdateLesson)
			lessonsGroup.DELETE("/:id", cfg.CourseHandler.DeleteLesson)
			lessonsGroup.POST("/:lessonId/complete", cfg.EnrollmentHandler.CompleteLesson)
		}

		// 5. User Enrollments & Progress
		studentGroup := api.Group("")
		studentGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService))
		{
			studentGroup.POST("/courses/:courseId/enroll", cfg.EnrollmentHandler.EnrollCourse)
			studentGroup.GET("/courses/enrolled", cfg.EnrollmentHandler.GetMyEnrollments)
			studentGroup.GET("/user/enrolled-courses", cfg.EnrollmentHandler.GetMyEnrollments)
			studentGroup.GET("/enrollments/:courseId/status", cfg.EnrollmentHandler.GetEnrollmentStatus)
		}

		// 6. Media Upload Group
		uploadGroup := api.Group("/upload")
		if cfg.UploadHandler != nil {
			uploadGroup.POST("/image", cfg.UploadHandler.UploadImage)
			uploadGroup.POST("/video", cfg.UploadHandler.UploadVideo)
			uploadGroup.POST("/pdf", cfg.UploadHandler.UploadPDF)
			uploadGroup.DELETE("/delete", cfg.UploadHandler.DeleteFile)
		}

		// 7. Chat Group
		chatGroup := api.Group("/conversations")
		chatGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService), middleware.BlockAdminChatMiddleware())
		{
			chatGroup.GET("", cfg.ChatHandler.GetConversations)
			chatGroup.GET("/:id/messages", cfg.ChatHandler.GetMessages)
			chatGroup.POST("/messages", cfg.ChatHandler.SendMessage)
			chatGroup.POST("/reactions", cfg.ChatHandler.SendReaction)
		}
	}

	return cfg.Engine
}
