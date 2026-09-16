package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"nexura-backend/internal/core/config"
	"nexura-backend/internal/core/server"
	"nexura-backend/internal/modules/admin"
	"nexura-backend/internal/modules/auth"
	"nexura-backend/internal/modules/certificate"
	"nexura-backend/internal/modules/chat"
	"nexura-backend/internal/modules/course"
	"nexura-backend/internal/modules/instructor"
	"nexura-backend/internal/modules/live"
	"nexura-backend/internal/modules/notification"
	"nexura-backend/internal/modules/payment"
	"nexura-backend/internal/modules/quiz"
	"nexura-backend/internal/modules/upload"
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
		if sqlBytes, err := os.ReadFile("db/migrations/000001_init_schema.up.sql"); err == nil {
			if _, err := db.Exec(string(sqlBytes)); err != nil {
				log.Printf("[WARNING] Auto-migration: %v", err)
			} else {
				log.Println("[Database] Schema migration applied.")
			}
		}
	}

	accessTTL := parseDuration(cfg.JWTAccessExpires, 7*24*time.Hour)
	refreshTTL := parseDuration(cfg.JWTRefreshExpires, 30*24*time.Hour)
	jwtService := jwt.NewJWTServiceWithRefresh(cfg.JWTSecret, cfg.JWTRefreshSecret, accessTTL, refreshTTL)

	userRepo := auth.NewUserRepository(db)
	courseRepo := course.NewCourseRepository(db)
	chatRepo := chat.NewChatRepository(db)
	payRepo := payment.NewRepository(db)

	authUsecase := auth.NewAuthUsecase(userRepo, jwtService)
	courseUsecase := course.NewCourseUsecase(courseRepo)
	chatUsecase := chat.NewChatUsecase(chatRepo, userRepo)

	authHandler := auth.NewAuthHandler(authUsecase)
	courseHandler := course.NewCourseHandler(courseUsecase)
	instructorHandler := instructor.NewInstructorHandler(db, courseRepo, payRepo)
	adminHandler := admin.NewAdminHandler(db, userRepo, courseRepo, payRepo)
	chatHandler := chat.NewChatHandler(chatUsecase)
	uploadHandler := upload.NewUploadHandler()
	paymentHandler := payment.NewHandler(payRepo)
	quizHandler := quiz.NewHandler(db)
	liveHandler := live.NewHandler(db)
	notifHandler := notification.NewHandler(db)
	certHandler := certificate.NewHandler(db)

	wsHub := chat.NewHub()
	go wsHub.Run()
	wsHandler := chat.NewWSHandler(wsHub, jwtService, chatUsecase)

	ioServer := chat.NewSocketServer(jwtService, chatUsecase)

	r := gin.Default()
	if ioServer != nil {
		r.GET("/socket.io/*any", gin.WrapH(ioServer))
		r.POST("/socket.io/*any", gin.WrapH(ioServer))
		go func() {
			if err := ioServer.Serve(); err != nil {
				log.Printf("[Socket.IO] %v", err)
			}
		}()
		defer ioServer.Close()
	}

	server.SetupServer(server.ServerConfig{
		Engine:            r,
		JWTService:        jwtService,
		ClientOrigin:      cfg.ClientOrigin,
		AuthHandler:       authHandler,
		CourseHandler:     courseHandler,
		InstructorHandler: instructorHandler,
		AdminHandler:      adminHandler,
		ChatHandler:       chatHandler,
		WSHandler:         wsHandler,
		UploadHandler:     uploadHandler,
		PaymentHandler:    paymentHandler,
		QuizHandler:       quizHandler,
		LiveHandler:       liveHandler,
		NotifHandler:      notifHandler,
		CertHandler:       certHandler,
	})

	log.Printf("[Nexura Hub] Server listening on http://localhost:%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	if strings.HasSuffix(s, "d") {
		n := strings.TrimSuffix(s, "d")
		if d, err := time.ParseDuration(n + "h"); err == nil {
			return d * 24
		}
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return fallback
}
