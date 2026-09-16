package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"nexura-backend/internal/core/config"
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

	log.Println("[Seeder] Applying schema...")
	migrationSQL, err := os.ReadFile("db/migrations/000001_init_schema.up.sql")
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}
	if _, err = db.ExecContext(ctx, string(migrationSQL)); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	pass, _ := hash.HashPassword("password123")
	adminID := uuid.MustParse("f659784d-1f7f-44f5-9e32-c1f7bb5aafc5")
	instructorID := uuid.MustParse("d4a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	student1 := uuid.MustParse("c7a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	student2 := uuid.MustParse("a1b2c3d4-1111-2222-3333-444455556666")
	courseID := uuid.MustParse("e4a7b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	adminCourseID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation, designation, phone, website)
		VALUES ($1,'Nexura','Admin','admin@nexurahub.com',$2,'admin','active',
			'https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=500',
			'Platform administrator','Platform Administrator','Super Admin','+8801700000000','https://nexurahub.com')
		ON CONFLICT (email) DO UPDATE SET password_hash=EXCLUDED.password_hash, role='admin', status='active'
	`, adminID, pass)
	if err != nil {
		log.Printf("admin seed: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation, designation, phone, website)
		VALUES ($1,'Tapas','Adhikary','tapas@nexurahub.com',$2,'instructor','active',
			'https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=500',
			'Senior Fullstack Engineer & Educator','Principal Software Architect','Senior Software Engineer & Educator','+8801712345678','https://tapasadhikary.com')
		ON CONFLICT (email) DO UPDATE SET password_hash=EXCLUDED.password_hash, role='instructor', status='active', designation=EXCLUDED.designation
	`, instructorID, pass)
	if err != nil {
		log.Printf("instructor seed: %v", err)
	}

	_, _ = db.ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation)
		VALUES ($1,'Karim','Rahman','student@nexurahub.com',$2,'student','active',
			'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=500','Aspiring fullstack developer','CS Student')
		ON CONFLICT (email) DO UPDATE SET password_hash=EXCLUDED.password_hash
	`, student1, pass)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status, avatar)
		VALUES ($1,'Sadia','Rahman','sadia@nexurahub.com',$2,'student','active',
			'https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=500')
		ON CONFLICT (email) DO UPDATE SET password_hash=EXCLUDED.password_hash
	`, student2, pass)

	_, _ = db.ExecContext(ctx, `INSERT INTO wallets (id, owner_type, user_id, balance) VALUES ($1,'admin',$2,0) ON CONFLICT (owner_type, user_id) DO NOTHING`, uuid.New(), adminID)
	_, _ = db.ExecContext(ctx, `INSERT INTO wallets (id, owner_type, user_id, balance) VALUES ($1,'instructor',$2,0) ON CONFLICT (owner_type, user_id) DO NOTHING`, uuid.New(), instructorID)

	cats := []struct{ title, slug, thumb string }{
		{"Web Development", "web-development", "https://images.unsplash.com/photo-1633356122544-f134324a6cee?w=800"},
		{"Design", "design", "https://images.unsplash.com/photo-1561070791-2526d30994b5?w=800"},
		{"Data Science", "data-science", "https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=800"},
		{"Mobile", "mobile", "https://images.unsplash.com/photo-1512941937669-90a1b58e7e9c?w=800"},
		{"Marketing", "marketing", "https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=800"},
		{"Business", "business", "https://images.unsplash.com/photo-1507679799987-c73779587ccf?w=800"},
		{"Photography", "photography", "https://images.unsplash.com/photo-1516035069371-29a1b244cc32?w=800"},
		{"Music", "music", "https://images.unsplash.com/photo-1511379938547-c1f69419868d?w=800"},
	}
	var webDevID uuid.UUID
	for _, cat := range cats {
		id := uuid.New()
		_, err := db.ExecContext(ctx, `
			INSERT INTO categories (id, title, slug, thumbnail) VALUES ($1,$2,$3,$4)
			ON CONFLICT (slug) DO UPDATE SET title=EXCLUDED.title, thumbnail=EXCLUDED.thumbnail
		`, id, cat.title, cat.slug, cat.thumb)
		if err != nil {
			log.Printf("category %s: %v", cat.slug, err)
		}
		if cat.slug == "web-development" {
			_ = db.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug='web-development'`).Scan(&webDevID)
		}
	}

	points := pq.StringArray{"Build production-grade apps", "Master React 19 concurrent features", "Design Go clean architecture APIs"}
	_, _ = db.ExecContext(ctx, `
		INSERT INTO courses (id, title, slug, subtitle, description, category_id, instructor_id, creator_type, thumbnail, price, discount_price, is_published, is_featured, learning_points)
		VALUES ($1,'Reactive Accelerator','reactive-accelerator','Master React 19 & Redux Toolkit with Go',
			'Comprehensive fullstack engineering course with real-world programming projects.',
			$2,$3,'instructor','https://images.unsplash.com/photo-1633356122544-f134324a6cee?w=800',4999,3999,true,true,$4)
		ON CONFLICT (slug) DO UPDATE SET is_published=true, is_featured=true, price=4999, discount_price=3999
	`, courseID, webDevID, instructorID, points)

	_, _ = db.ExecContext(ctx, `
		INSERT INTO courses (id, title, slug, subtitle, description, category_id, instructor_id, creator_type, thumbnail, price, is_published, is_featured, learning_points)
		VALUES ($1,'Nexura Platform Masterclass','nexura-platform-masterclass','Learn the LMS from the inside',
			'Admin-owned course covering how Nexura Hub works.',$2,$3,'admin','https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800',2000,true,false,$4)
		ON CONFLICT (slug) DO NOTHING
	`, adminCourseID, webDevID, adminID, pq.StringArray{"Understand dummy wallets", "Read admin revenue"})

	mod1 := uuid.MustParse("b1a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	mod2 := uuid.MustParse("b1a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e70")
	_, _ = db.ExecContext(ctx, `INSERT INTO modules (id, course_id, title, description, position, is_published) VALUES ($1,$2,'Module 1: Frontend Foundations','React 19 basics',1,true) ON CONFLICT DO NOTHING`, mod1, courseID)
	_, _ = db.ExecContext(ctx, `INSERT INTO modules (id, course_id, title, description, position, is_published) VALUES ($1,$2,'Module 2: Backend APIs','Golang APIs',2,true) ON CONFLICT DO NOTHING`, mod2, courseID)

	lessons := []struct {
		id, mod uuid.UUID
		title, url, dur string
		free            bool
		pos             int
	}{
		{uuid.MustParse("b2a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f"), mod1, "Course Overview", "https://www.youtube.com/watch?v=8sXRyHI3bLw", "12:40", true, 1},
		{uuid.MustParse("b2a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e70"), mod1, "React 19 Server Components", "https://www.youtube.com/watch?v=TNhaISOUy4Q", "22:10", false, 2},
		{uuid.MustParse("b2a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e71"), mod1, "Redux Toolkit Query", "https://www.youtube.com/watch?v=9zySeP5vH9c", "18:05", false, 3},
		{uuid.MustParse("b3a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f"), mod2, "Golang Clean Architecture", "https://www.youtube.com/watch?v=YS4e4q9oBaU", "42:15", false, 1},
		{uuid.MustParse("b3a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e70"), mod2, "JWT Auth in Go", "https://www.youtube.com/watch?v=0Tu_0AcOL1U", "16:20", false, 2},
		{uuid.MustParse("b3a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e71"), mod2, "PostgreSQL & Transactions", "https://www.youtube.com/watch?v=qw--VYLpxG4", "25:00", false, 3},
	}
	for _, l := range lessons {
		_, _ = db.ExecContext(ctx, `
			INSERT INTO lessons (id, module_id, title, description, video_url, duration, is_free, is_published, position)
			VALUES ($1,$2,$3,$4,$5,$6,$7,true,$8) ON CONFLICT DO NOTHING
		`, l.id, l.mod, l.title, l.title, l.url, l.dur, l.free, l.pos)
	}
	_, _ = db.ExecContext(ctx, `INSERT INTO lesson_resources (id, lesson_id, title, type, url, size) VALUES ($1,$2,'React 19 CheatSheet.pdf','pdf','https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf','2.4 MB') ON CONFLICT DO NOTHING`,
		uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001"), lessons[0].id)

	_, _ = db.ExecContext(ctx, `
		INSERT INTO coupons (id, code, discount_type, discount_value, expiry_date, max_redemptions, is_active, created_by)
		VALUES ($1,'NEXURA20','percentage',20,$2,500,true,$3)
		ON CONFLICT (code) DO NOTHING
	`, uuid.New(), time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), adminID)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO coupons (id, code, discount_type, discount_value, expiry_date, max_redemptions, is_active, created_by)
		VALUES ($1,'FLAT500','flat',500,$2,200,true,$3)
		ON CONFLICT (code) DO NOTHING
	`, uuid.New(), time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), adminID)

	groupID := uuid.MustParse("cccccccc-dddd-eeee-ffff-000000000001")
	_, _ = db.ExecContext(ctx, `
		INSERT INTO conversations (id, type, name, course_id, instructor_id)
		VALUES ($1,'group','Reactive Accelerator Community Group',$2,$3)
		ON CONFLICT DO NOTHING
	`, groupID, courseID, instructorID)
	_, _ = db.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, groupID, instructorID)
	_, _ = db.ExecContext(ctx, `INSERT INTO messages (id, conversation_id, sender_id, content) VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
		uuid.New(), groupID, instructorID, "Welcome to the official group chat for Reactive Accelerator!")

	log.Println("[Seeder] Done.")
	log.Println("🔑 Admin:      admin@nexurahub.com / password123")
	log.Println("🔑 Instructor: tapas@nexurahub.com / password123")
	log.Println("🔑 Student:    student@nexurahub.com / password123")
	log.Println("🔑 Student 2:  sadia@nexurahub.com / password123")
	log.Println("🎟️  Coupons:   NEXURA20 (20%)  FLAT500 (৳500)")
}
