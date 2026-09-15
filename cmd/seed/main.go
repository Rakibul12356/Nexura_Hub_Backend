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

	adminID := uuid.MustParse("f659784d-1f7f-44f5-9e32-c1f7bb5aafc5")
	instructorID := uuid.MustParse("d4a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	studentID := uuid.MustParse("c7a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")

	// 1. Admin Account Seeding
	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation, phone, website)
		VALUES ($1, 'Nexura', 'Admin', 'NexuraHubAdmin@gmail.com', $2, 'admin', 'active',
			'https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=500',
			'Super Admin & Platform Administrator for Nexura Hub.',
			'Platform Administrator', '+8801700000000', 'https://nexurahub.com')
		ON CONFLICT (email) DO UPDATE 
		SET password_hash = EXCLUDED.password_hash, role = 'admin', status = 'active',
		    avatar = EXCLUDED.avatar, bio = EXCLUDED.bio, occupation = EXCLUDED.occupation
	`, adminID, pass123)
	if err != nil {
		log.Printf("Error seeding admin: %v", err)
	}

	// 2. Instructor Account Seeding
	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation, phone, website)
		VALUES ($1, 'Tapas', 'Adhikary', 'instructor@nexurahub.com', $2, 'instructor', 'active',
			'https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=500',
			'Senior Fullstack Engineer & Tech Lead with 10+ years of experience in Go, React & Cloud Native Systems.',
			'Principal Software Architect', '+8801712345678', 'https://tapasadhikary.com')
		ON CONFLICT (email) DO UPDATE 
		SET password_hash = EXCLUDED.password_hash, role = 'instructor', status = 'active',
		    avatar = EXCLUDED.avatar, bio = EXCLUDED.bio, occupation = EXCLUDED.occupation
	`, instructorID, pass123)
	if err != nil {
		log.Printf("Error seeding instructor: %v", err)
	}

	// 3. Student Account Seeding
	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation, phone, website)
		VALUES ($1, 'Karim', 'Rahman', 'student@nexurahub.com', $2, 'student', 'active',
			'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=500',
			'Aspiring Fullstack Developer learning React 19, TypeScript and Golang microservices.',
			'CS Student & Junior Developer', '+8801812345678', 'https://github.com/karim-rahman')
		ON CONFLICT (email) DO UPDATE 
		SET password_hash = EXCLUDED.password_hash, role = 'student', status = 'active',
		    avatar = EXCLUDED.avatar, bio = EXCLUDED.bio, occupation = EXCLUDED.occupation
	`, studentID, pass123)
	if err != nil {
		log.Printf("Error seeding student: %v", err)
	}

	// 4. Categories Seeding
	var catID int
	err = db.QueryRowContext(ctx, `
		INSERT INTO categories (title, slug, thumbnail)
		VALUES ('Web Development', 'web-dev', 'https://images.unsplash.com/photo-1633356122544-f134324a6cee?w=800')
		ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title
		RETURNING id
	`).Scan(&catID)
	if err != nil {
		_ = db.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug = 'web-dev'`).Scan(&catID)
	}

	// 5. Course Seeding with YouTube Programming Tutorials
	courseID := uuid.MustParse("e4a7b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	learningPoints, _ := json.Marshal([]string{
		"Master React 19 concurrent features & server actions",
		"Build scalable Redux Toolkit global state architecture",
		"Design high-performance Golang clean architecture backend APIs",
	})

	_, _ = db.ExecContext(ctx, `
		INSERT INTO courses (id, title, slug, subtitle, description, category_id, instructor_id, thumbnail, price, discount_price, is_published, learning_points)
		VALUES ($1, 'Reactive Accelerator - Fullstack Go & React 19', 'reactive-accelerator', 'Master React 19 & Redux Toolkit with Go', 'Comprehensive fullstack engineering course with real-world programming projects.', $2, $3, 'https://images.unsplash.com/photo-1633356122544-f134324a6cee?w=800', 4500.00, 3500.00, true, $4)
		ON CONFLICT (slug) DO NOTHING
	`, courseID, catID, instructorID, learningPoints)

	// 6. Modules Seeding
	moduleID := uuid.MustParse("b1a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	_, _ = db.ExecContext(ctx, `
		INSERT INTO modules (id, course_id, title, description, position, is_published)
		VALUES ($1, $2, 'Module 1: Modern Frontend & Backend Architecture', 'Getting started with React 19, TypeScript and Golang.', 1, true)
		ON CONFLICT DO NOTHING
	`, moduleID, courseID)

	// 7. Lessons Seeding with Real YouTube Programming Links
	lesson1ID := uuid.MustParse("b2a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	resources1, _ := json.Marshal([]course.LessonResource{
		{ID: "res-1", Title: "React 19 CheatSheet & Architecture Guide.pdf", URL: "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf", Type: "pdf", Size: "2.4 MB"},
	})
	_, _ = db.ExecContext(ctx, `
		INSERT INTO lessons (id, module_id, title, description, video_url, duration, is_free, is_published, position, resources)
		VALUES ($1, $2, 'Lesson 1.1: React 19 Full Course & Server Components', 'Complete breakdown of React 19 features.', 'https://www.youtube.com/watch?v=8sXRyHI3bLw', '25:30', true, true, 1, $3)
		ON CONFLICT DO NOTHING
	`, lesson1ID, moduleID, resources1)

	lesson2ID := uuid.MustParse("b3a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	_, _ = db.ExecContext(ctx, `
		INSERT INTO lessons (id, module_id, title, description, video_url, duration, is_free, is_published, position, resources)
		VALUES ($1, $2, 'Lesson 1.2: Golang Backend API & Clean Architecture', 'Building production-grade Go REST APIs.', 'https://www.youtube.com/watch?v=YS4e4q9oBaU', '42:15', false, true, 2, $3)
		ON CONFLICT DO NOTHING
	`, lesson2ID, moduleID, resources1)

	// 8. Enrollments Seeding
	enrollmentID := uuid.New()
	_, _ = db.ExecContext(ctx, `
		INSERT INTO enrollments (id, student_id, course_id, amount_paid, payment_method, payment_status, completed_lessons, progress_percentage)
		VALUES ($1, $2, $3, 3500.00, 'bKash', 'completed', '[]', 50)
		ON CONFLICT DO NOTHING
	`, enrollmentID, studentID, courseID)

	// 9. Conversations Seeding
	convID := uuid.New()
	_, _ = db.ExecContext(ctx, `
		INSERT INTO conversations (id, type, name, avatar, course_id, instructor_id)
		VALUES ($1, 'group', 'Reactive Accelerator Discussion Group', 'https://images.unsplash.com/photo-1633356122544-f134324a6cee?w=800', $2, $3)
		ON CONFLICT DO NOTHING
	`, convID, courseID, instructorID)

	_, _ = db.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1, $2), ($1, $3) ON CONFLICT DO NOTHING`, convID, instructorID, studentID)

	log.Println("[Seeder] Database successfully populated with real accounts & programming video courses!")
	log.Println("🔑 Admin: NexuraHubAdmin@gmail.com / password123")
	log.Println("🔑 Instructor: instructor@nexurahub.com / password123")
	log.Println("🔑 Student: student@nexurahub.com / password123")
}
