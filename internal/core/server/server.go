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
}

func SetupServer(cfg ServerConfig) *gin.Engine {
	cfg.Engine.Use(middleware.CORSMiddleware())

	cfg.Engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "Nexura Hub API Server"})
	})

	cfg.Engine.GET("/ws", cfg.WSHandler.ServeWS)

	api := cfg.Engine.Group("/api/v1")
	{
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", cfg.AuthHandler.Register)
			authGroup.POST("/login", cfg.AuthHandler.Login)
			authGroup.GET("/me", middleware.JWTAuthMiddleware(cfg.JWTService), cfg.AuthHandler.GetMe)
		}

		courseGroup := api.Group("/courses")
		{
			courseGroup.GET("", cfg.CourseHandler.ListCourses)
			courseGroup.GET("/categories", cfg.CourseHandler.GetCategories)
			courseGroup.GET("/:slug", cfg.CourseHandler.GetCourseBySlug)
		}

		studentGroup := api.Group("")
		studentGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService))
		{
			studentGroup.POST("/courses/:courseId/enroll", cfg.EnrollmentHandler.EnrollCourse)
			studentGroup.GET("/courses/enrolled", cfg.EnrollmentHandler.GetMyEnrollments)
			studentGroup.POST("/lessons/:lessonId/complete", cfg.EnrollmentHandler.CompleteLesson)
		}

		instructorGroup := api.Group("/instructor")
		instructorGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService), middleware.RequireRole("instructor", "admin"))
		{
			instructorGroup.GET("/dashboard/stats", cfg.InstructorHandler.GetDashboardStats)
			instructorGroup.POST("/courses", cfg.CourseHandler.CreateCourse)
		}

		adminGroup := api.Group("/admin")
		adminGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTService), middleware.RequireRole("admin"))
		{
			adminGroup.GET("/overview", cfg.AdminHandler.GetOverviewStats)
			adminGroup.PATCH("/courses/:id/approve", cfg.AdminHandler.ApproveCourse)
			adminGroup.PATCH("/users/:id/status", cfg.AdminHandler.UpdateUserStatus)
		}

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
