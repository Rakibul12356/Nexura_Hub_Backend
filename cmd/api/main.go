// cmd/api/main.go
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"nexura-backend/internal/core/config"
	"nexura-backend/internal/core/server"
	"nexura-backend/internal/modules/admin"
	"nexura-backend/internal/modules/auth"
	"nexura-backend/internal/modules/chat"
	"nexura-backend/internal/modules/course"
	"nexura-backend/internal/modules/enrollment"
	"nexura-backend/internal/modules/instructor"
	"nexura-backend/pkg/database"
	"nexura-backend/pkg/jwt"
)

func main() {
	cfg := config.LoadConfig()

	log.Printf("[Nexura Hub] Starting backend service on port %s (env: %s)...", cfg.Port, cfg.Environment)

	db, err := database.NewPostgresDB(cfg.DBUrl)
	if err != nil {
		log.Printf("[WARNING] PostgreSQL connection failed: %v.", err)
	} else {
		defer db.Close()
		log.Println("[Database] PostgreSQL connection pool established.")
	}

	jwtService := jwt.NewJWTService(cfg.JWTSecret)

	// Repositories
	userRepo := auth.NewUserRepository(db)
	courseRepo := course.NewCourseRepository(db)
	enrollmentRepo := enrollment.NewEnrollmentRepository(db)
	instructorRepo := instructor.NewInstructorRepository(db)
	adminRepo := admin.NewAdminRepository(db)
	chatRepo := chat.NewChatRepository(db)

	// Usecases
	authUsecase := auth.NewAuthUsecase(userRepo, jwtService)
	courseUsecase := course.NewCourseUsecase(courseRepo)
	enrollmentUsecase := enrollment.NewEnrollmentUsecase(enrollmentRepo)
	instructorUsecase := instructor.NewInstructorUsecase(instructorRepo)
	adminUsecase := admin.NewAdminUsecase(adminRepo, courseRepo, userRepo)
	chatUsecase := chat.NewChatUsecase(chatRepo, userRepo)

	// Handlers
	authHandler := auth.NewAuthHandler(authUsecase)
	courseHandler := course.NewCourseHandler(courseUsecase)
	enrollmentHandler := enrollment.NewEnrollmentHandler(enrollmentUsecase)
	instructorHandler := instructor.NewInstructorHandler(instructorUsecase)
	adminHandler := admin.NewAdminHandler(adminUsecase)
	chatHandler := chat.NewChatHandler(chatUsecase)

	// WebSocket Hub & Handler
	wsHub := chat.NewHub()
	go wsHub.Run()
	wsHandler := chat.NewWSHandler(wsHub, jwtService, chatUsecase)

	// Router Setup
	r := gin.Default()
	server.SetupServer(server.ServerConfig{
		Engine:            r,
		JWTService:        jwtService,
		AuthHandler:       authHandler,
		CourseHandler:     courseHandler,
		EnrollmentHandler: enrollmentHandler,
		InstructorHandler: instructorHandler,
		AdminHandler:      adminHandler,
		ChatHandler:       chatHandler,
		WSHandler:         wsHandler,
	})

	log.Printf("[Nexura Hub] Server listening on http://localhost:%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
