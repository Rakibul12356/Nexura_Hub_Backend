// cmd/seed/main.go
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/google/uuid"
	"nexura-backend/internal/core/config"
	"nexura-backend/internal/modules/course"
	"nexura-backend/pkg/database"
	"nexura-backend/pkg/hash"
)

func main() {
	cfg := config.LoadConfig()
	db, err := database.NewPostgresDB(cfg.DBUrl)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	log.Println("[Seeder] Executing database migration schema...")
	migrationSQL, err := os.ReadFile("db/migrations/000001_init_schema.up.sql")
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}

	_, err = db.ExecContext(ctx, string(migrationSQL))
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("[Seeder] Migration completed successfully.")

	pass123, _ := hash.HashPassword("password123")

	adminID := uuid.New()
	instructorID := uuid.New()
	studentID := uuid.New()

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status)
		VALUES ($1, 'Nexura', 'Admin', 'NexuraHubAdmin@gmail.com', $2, 'admin', 'active')
		ON CONFLICT (email) DO UPDATE 
		SET password_hash = EXCLUDED.password_hash, role = 'admin', status = 'active'
	`, adminID, pass123)
	if err != nil {
		log.Printf("Error seeding admin: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status)
		VALUES ($1, 'Tapas', 'Adhikary', 'instructor@nexurahub.com', $2, 'instructor', 'active')
		ON CONFLICT (email) DO UPDATE 
		SET password_hash = EXCLUDED.password_hash, role = 'instructor', status = 'active'
	`, instructorID, pass123)
	if err != nil {
		log.Printf("Error seeding instructor: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status)
		VALUES ($1, 'John', 'Student', 'student@nexurahub.com', $2, 'student', 'active')
		ON CONFLICT (email) DO UPDATE 
		SET password_hash = EXCLUDED.password_hash, role = 'student', status = 'active'
	`, studentID, pass123)
	if err != nil {
		log.Printf("Error seeding student: %v", err)
	}

	_ = db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = 'NexuraHubAdmin@gmail.com'`).Scan(&adminID)
	_ = db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = 'instructor@nexurahub.com'`).Scan(&instructorID)
	_ = db.QueryRowContext(ctx, `SELECT id FROM users WHERE email = 'student@nexurahub.com'`).Scan(&studentID)

	var catID int
	err = db.QueryRowContext(ctx, `
		INSERT INTO categories (title, slug, thumbnail)
		VALUES ('Web Development', 'web-dev', 'https://images.unsplash.com/photo-1633356122544-f134324a6cee')
		ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title
		RETURNING id
	`).Scan(&catID)
	if err != nil {
		_ = db.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug = 'web-dev'`).Scan(&catID)
	}

	courseID := uuid.New()
	learningPoints, _ := json.Marshal([]string{
		"Master React 19 concurrent features & server actions",
		"Build scalable Redux Toolkit global state architecture",
		"Design high-performance Golang clean architecture backend APIs",
	})

	_, _ = db.ExecContext(ctx, `
		INSERT INTO courses (id, title, slug, subtitle, description, category_id, instructor_id, thumbnail, price, discount_price, is_published, learning_points)
		VALUES ($1, 'Reactive Accelerator', 'reactive-accelerator', 'Master React 19 & Redux Toolkit with Go', 'Comprehensive fullstack engineering course.', $2, $3, 'https://images.unsplash.com/photo-1633356122544-f134324a6cee', 4500.00, 3500.00, true, $4)
		ON CONFLICT (slug) DO NOTHING
	`, courseID, catID, instructorID, learningPoints)

	_ = db.QueryRowContext(ctx, `SELECT id FROM courses WHERE slug = 'reactive-accelerator'`).Scan(&courseID)

	moduleID := uuid.New()
	_, _ = db.ExecContext(ctx, `
		INSERT INTO modules (id, course_id, title, description, position, is_published)
		VALUES ($1, $2, 'Module 1: Modern Frontend Architecture', 'Getting started with React 19 and state management.', 1, true)
		ON CONFLICT DO NOTHING
	`, moduleID, courseID)

	lessonID := uuid.New()
	resources, _ := json.Marshal([]course.LessonResource{
		{ID: "res-1", Title: "React 19 CheatSheet.pdf", URL: "https://example.com/sheet.pdf", Type: "pdf", Size: "2.4 MB"},
	})
	_, _ = db.ExecContext(ctx, `
		INSERT INTO lessons (id, module_id, title, description, video_url, duration, is_free, is_published, position, resources)
		VALUES ($1, $2, 'Lesson 1.1: Introduction to React 19', 'Overview of new React 19 APIs.', 'https://www.youtube.com/watch?v=dQw4w9WgXcQ', '12:45', true, true, 1, $3)
		ON CONFLICT DO NOTHING
	`, lessonID, moduleID, resources)

	convID := uuid.New()
	_, _ = db.ExecContext(ctx, `
		INSERT INTO conversations (id, type, name, avatar, course_id, instructor_id)
		VALUES ($1, 'group', 'Reactive Accelerator Discussion Group', 'https://images.unsplash.com/photo-1633356122544-f134324a6cee', $2, $3)
		ON CONFLICT DO NOTHING
	`, convID, courseID, instructorID)

	_ = db.QueryRowContext(ctx, `SELECT id FROM conversations WHERE course_id = $1 LIMIT 1`, courseID).Scan(&convID)

	_, _ = db.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1, $2), ($1, $3) ON CONFLICT DO NOTHING`, convID, instructorID, studentID)

	log.Println("[Seeder] Database successfully updated! Admin: NexuraHubAdmin@gmail.com / password123")
}
