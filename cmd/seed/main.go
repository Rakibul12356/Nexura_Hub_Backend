package main

import (
	"context"
	"database/sql"
	"log"
	"strings"
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
	database.RepairCompat(db)

	pass, err := hash.HashPassword("password123")
	if err != nil {
		log.Fatal(err)
	}

	adminID := uuid.MustParse("f659784d-1f7f-44f5-9e32-c1f7bb5aafc5")
	instructorID := uuid.MustParse("d4a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	student1 := uuid.MustParse("c7a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	student2 := uuid.MustParse("a1b2c3d4-1111-2222-3333-444455556666")

	keepEmails := map[string]uuid.UUID{
		"admin@nexurahub.com":    adminID,
		"tapas@nexurahub.com":    instructorID,
		"student@nexurahub.com":  student1,
		"sadia@nexurahub.com":    student2,
	}
	removeEmails := []string{
		"nexurahubadmin@gmail.com",
		"instructor@nexurahub.com",
		"john@example.com",
		"jagy@mailinator.com",
	}

	upsertUser(ctx, db, adminID, "Nexura", "Admin", "admin@nexurahub.com", pass, "admin",
		"https://images.unsplash.com/photo-1535713875002-d1d0cf377fde?w=500",
		"Platform administrator", "Platform Administrator", "Super Admin", "+8801700000000", "https://nexurahub.com")
	upsertUser(ctx, db, instructorID, "Tapas", "Adhikary", "tapas@nexurahub.com", pass, "instructor",
		"https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=500",
		"Senior Fullstack Engineer & Educator", "Principal Software Architect", "Senior Software Engineer & Educator", "+8801712345678", "https://tapasadhikary.com")
	upsertUser(ctx, db, student1, "Karim", "Rahman", "student@nexurahub.com", pass, "student",
		"https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=500",
		"Aspiring fullstack developer", "CS Student", "", "+8801811111111", "")
	upsertUser(ctx, db, student2, "Sadia", "Rahman", "sadia@nexurahub.com", pass, "student",
		"https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=500",
		"Frontend learner", "Student", "", "+8801822222222", "")

	adminID = mustUserID(ctx, db, "admin@nexurahub.com")
	instructorID = mustUserID(ctx, db, "tapas@nexurahub.com")
	student1 = mustUserID(ctx, db, "student@nexurahub.com")
	student2 = mustUserID(ctx, db, "sadia@nexurahub.com")

	exec(ctx, db, `UPDATE courses SET instructor_id=$1 WHERE instructor_id IN (SELECT id FROM users WHERE lower(email)=ANY($2))`,
		instructorID, pq.Array(removeEmails))
	exec(ctx, db, `UPDATE live_classes SET instructor_id=$1 WHERE instructor_id IN (SELECT id FROM users WHERE lower(email)=ANY($2))`,
		instructorID, pq.Array(removeEmails))
	exec(ctx, db, `UPDATE quiz_sets SET instructor_id=$1 WHERE instructor_id IN (SELECT id FROM users WHERE lower(email)=ANY($2))`,
		instructorID, pq.Array(removeEmails))
	exec(ctx, db, `UPDATE conversations SET instructor_id=$1 WHERE instructor_id IN (SELECT id FROM users WHERE lower(email)=ANY($2))`,
		instructorID, pq.Array(removeEmails))
	exec(ctx, db, `UPDATE coupons SET created_by=$1 WHERE created_by IN (SELECT id FROM users WHERE lower(email)=ANY($2))`,
		adminID, pq.Array(removeEmails))

	oldIDs := emailsToIDs(ctx, db, removeEmails)
	for _, id := range oldIDs {
		wipeUser(ctx, db, id, instructorID, student1)
	}

	webDevID := ensureCategories(ctx, db)
	courseID := ensureCourse(ctx, db, instructorID, webDevID)
	adminCourseID := ensureAdminCourse(ctx, db, adminID, webDevID)
	ensureCurriculum(ctx, db, courseID)
	ensureWallets(ctx, db, adminID, instructorID)
	ensureCoupons(ctx, db, adminID)
	ensureEnrollment(ctx, db, student1, student2, courseID, instructorID, adminID)
	ensureReview(ctx, db, student2, courseID)
	ensureLive(ctx, db, instructorID, courseID)
	ensureQuiz(ctx, db, instructorID)
	ensureChat(ctx, db, courseID, instructorID, student1)
	ensureNotifications(ctx, db, adminID, instructorID, student1, student2, courseID)

	_ = adminCourseID
	_ = keepEmails
	log.Println("[Seeder] Demo accounts reset.")
	log.Println("🔑 Admin:      admin@nexurahub.com / password123")
	log.Println("🔑 Instructor: tapas@nexurahub.com / password123")
	log.Println("🔑 Student:    student@nexurahub.com / password123")
	log.Println("🔑 Student 2:  sadia@nexurahub.com / password123")
}

func upsertUser(ctx context.Context, db *sql.DB, id uuid.UUID, first, last, email, pass, role, avatar, bio, occupation, designation, phone, website string) {
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, first_name, last_name, email, password_hash, role, status, avatar, bio, occupation, designation, phone, website, deleted_at)
		VALUES ($1,$2,$3,$4,$5,$6,'active',$7,$8,$9,NULLIF($10,''),NULLIF($11,''),NULLIF($12,''), NULL)
		ON CONFLICT (email) DO UPDATE SET
			password_hash=EXCLUDED.password_hash,
			role=EXCLUDED.role,
			status='active',
			first_name=EXCLUDED.first_name,
			last_name=EXCLUDED.last_name,
			avatar=EXCLUDED.avatar,
			bio=EXCLUDED.bio,
			occupation=EXCLUDED.occupation,
			designation=EXCLUDED.designation,
			phone=EXCLUDED.phone,
			website=EXCLUDED.website,
			deleted_at=NULL
	`, id, first, last, strings.ToLower(email), pass, role, avatar, bio, occupation, designation, phone, website)
	if err != nil {
		log.Fatalf("upsert %s: %v", email, err)
	}
}

func mustUserID(ctx context.Context, db *sql.DB, email string) uuid.UUID {
	var id uuid.UUID
	if err := db.QueryRowContext(ctx, `SELECT id FROM users WHERE lower(email)=$1`, strings.ToLower(email)).Scan(&id); err != nil {
		log.Fatalf("lookup %s: %v", email, err)
	}
	return id
}

func emailsToIDs(ctx context.Context, db *sql.DB, emails []string) []uuid.UUID {
	rows, err := db.QueryContext(ctx, `SELECT id FROM users WHERE lower(email)=ANY($1)`, pq.Array(emails))
	if err != nil {
		log.Printf("list old users: %v", err)
		return nil
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		_ = rows.Scan(&id)
		out = append(out, id)
	}
	return out
}

func wipeUser(ctx context.Context, db *sql.DB, id, fallbackInstructor, fallbackStudent uuid.UUID) {
	exec(ctx, db, `DELETE FROM refresh_tokens WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM password_resets WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM email_verifications WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM notifications WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM message_reactions WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM conversation_members WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM messages WHERE sender_id=$1`, id)
	exec(ctx, db, `DELETE FROM direct_pairs WHERE user_a=$1 OR user_b=$1`, id)
	exec(ctx, db, `DELETE FROM enrollments WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM coupon_redemptions WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM payments WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM reviews WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM certificates WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM lesson_notes WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM lesson_progress WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM quiz_attempts WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM discussion_votes WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM discussion_replies WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM discussions WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM wallet_ledger WHERE wallet_id IN (SELECT id FROM wallets WHERE user_id=$1)`, id)
	exec(ctx, db, `DELETE FROM wallets WHERE user_id=$1`, id)
	exec(ctx, db, `DELETE FROM transactions WHERE student_id=$1 OR instructor_id=$1`, id)
	exec(ctx, db, `UPDATE live_classes SET instructor_id=$2 WHERE instructor_id=$1`, id, fallbackInstructor)
	exec(ctx, db, `UPDATE courses SET instructor_id=$2 WHERE instructor_id=$1`, id, fallbackInstructor)
	exec(ctx, db, `DELETE FROM users WHERE id=$1`, id)
}

func ensureCategories(ctx context.Context, db *sql.DB) uuid.UUID {
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
	for _, cat := range cats {
		exec(ctx, db, `
			INSERT INTO categories (id, title, slug, thumbnail) VALUES ($1,$2,$3,$4)
			ON CONFLICT (slug) DO UPDATE SET title=EXCLUDED.title, thumbnail=EXCLUDED.thumbnail
		`, uuid.New(), cat.title, cat.slug, cat.thumb)
	}
	var id uuid.UUID
	_ = db.QueryRowContext(ctx, `SELECT id FROM categories WHERE slug IN ('web-development','web-dev') ORDER BY slug DESC LIMIT 1`).Scan(&id)
	if id == uuid.Nil {
		log.Fatal("web development category missing")
	}
	return id
}

func ensureCourse(ctx context.Context, db *sql.DB, instructorID, catID uuid.UUID) uuid.UUID {
	points := pq.StringArray{"Build production-grade apps", "Master React 19 concurrent features", "Design Go clean architecture APIs"}
	var id uuid.UUID
	err := db.QueryRowContext(ctx, `SELECT id FROM courses WHERE slug='reactive-accelerator'`).Scan(&id)
	if err != nil {
		id = uuid.MustParse("e4a7b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
		exec(ctx, db, `
			INSERT INTO courses (id, title, slug, subtitle, description, category_id, instructor_id, creator_type, thumbnail, price, discount_price, is_published, is_featured, learning_points, deleted_at)
			VALUES ($1,'Reactive Accelerator','reactive-accelerator','Master React 19 & Redux Toolkit with Go',
				'Comprehensive fullstack engineering course with real-world programming projects.',
				$2,$3,'instructor','https://images.unsplash.com/photo-1633356122544-f134324a6cee?w=800',4999,3999,true,true,$4,NULL)
		`, id, catID, instructorID, points)
	} else {
		exec(ctx, db, `
			UPDATE courses SET instructor_id=$1, category_id=$2, is_published=true, is_featured=true,
				price=4999, discount_price=3999, learning_points=$3, deleted_at=NULL, creator_type='instructor',
				title='Reactive Accelerator', subtitle='Master React 19 & Redux Toolkit with Go'
			WHERE id=$4
		`, instructorID, catID, points, id)
	}
	return id
}

func ensureAdminCourse(ctx context.Context, db *sql.DB, adminID, catID uuid.UUID) uuid.UUID {
	var id uuid.UUID
	err := db.QueryRowContext(ctx, `SELECT id FROM courses WHERE slug='nexura-platform-masterclass'`).Scan(&id)
	if err != nil {
		id = uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
		exec(ctx, db, `
			INSERT INTO courses (id, title, slug, subtitle, description, category_id, instructor_id, creator_type, thumbnail, price, is_published, is_featured, learning_points)
			VALUES ($1,'Nexura Platform Masterclass','nexura-platform-masterclass','Learn the LMS from the inside',
				'Admin-owned course covering how Nexura Hub works.',$2,$3,'admin','https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800',0,true,false,$4)
		`, id, catID, adminID, pq.StringArray{"Understand dummy wallets", "Read admin revenue"})
	} else {
		exec(ctx, db, `UPDATE courses SET instructor_id=$1, price=0, is_published=true, creator_type='admin' WHERE id=$2`, adminID, id)
	}
	return id
}

func ensureCurriculum(ctx context.Context, db *sql.DB, courseID uuid.UUID) {
	mod1 := uuid.MustParse("b1a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e6f")
	mod2 := uuid.MustParse("b1a8b9f1-3d2e-4b5a-9f8e-1a2b3c4d5e70")
	exec(ctx, db, `INSERT INTO modules (id, course_id, title, description, position, is_published) VALUES ($1,$2,'Module 1: Frontend Foundations','React 19 basics',1,true) ON CONFLICT (id) DO UPDATE SET course_id=EXCLUDED.course_id, title=EXCLUDED.title, is_published=true`, mod1, courseID)
	exec(ctx, db, `INSERT INTO modules (id, course_id, title, description, position, is_published) VALUES ($1,$2,'Module 2: Backend APIs','Golang APIs',2,true) ON CONFLICT (id) DO UPDATE SET course_id=EXCLUDED.course_id, title=EXCLUDED.title, is_published=true`, mod2, courseID)
	lessons := []struct {
		id, mod         uuid.UUID
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
		exec(ctx, db, `
			INSERT INTO lessons (id, module_id, title, description, video_url, duration, is_free, is_published, position)
			VALUES ($1,$2,$3,$4,$5,$6,$7,true,$8)
			ON CONFLICT (id) DO UPDATE SET title=EXCLUDED.title, video_url=EXCLUDED.video_url, is_published=true
		`, l.id, l.mod, l.title, l.title, l.url, l.dur, l.free, l.pos)
	}
	exec(ctx, db, `INSERT INTO lesson_resources (id, lesson_id, title, type, url, size) VALUES ($1,$2,'React 19 CheatSheet.pdf','pdf','https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf','2.4 MB') ON CONFLICT (id) DO NOTHING`,
		uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001"), lessons[0].id)
}

func ensureWallets(ctx context.Context, db *sql.DB, adminID, instructorID uuid.UUID) {
	exec(ctx, db, `INSERT INTO wallets (id, owner_type, user_id, balance) VALUES ($1,'admin',$2,199.96) ON CONFLICT (owner_type, user_id) DO UPDATE SET balance=199.96`, uuid.New(), adminID)
	exec(ctx, db, `INSERT INTO wallets (id, owner_type, user_id, balance) VALUES ($1,'instructor',$2,3799.24) ON CONFLICT (owner_type, user_id) DO UPDATE SET balance=3799.24`, uuid.New(), instructorID)
}

func ensureCoupons(ctx context.Context, db *sql.DB, adminID uuid.UUID) {
	exp := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	exec(ctx, db, `
		INSERT INTO coupons (id, code, discount_type, discount_value, expiry_date, max_redemptions, is_active, created_by)
		VALUES ($1,'NEXURA20','percentage',20,$2,500,true,$3)
		ON CONFLICT (code) DO UPDATE SET is_active=true, discount_value=20
	`, uuid.New(), exp, adminID)
	exec(ctx, db, `
		INSERT INTO coupons (id, code, discount_type, discount_value, expiry_date, max_redemptions, is_active, created_by)
		VALUES ($1,'FLAT500','flat',500,$2,200,true,$3)
		ON CONFLICT (code) DO UPDATE SET is_active=true, discount_value=500
	`, uuid.New(), exp, adminID)
}

func ensureEnrollment(ctx context.Context, db *sql.DB, student1, student2, courseID, instructorID, adminID uuid.UUID) {
	payID := uuid.New()
	exec(ctx, db, `
		INSERT INTO payments (id, public_txn_id, user_id, course_id, original_price, discount_amount, amount_paid, gateway, gateway_label, is_dummy, status, paid_at)
		VALUES ($1,'TXN-DUMMY-18402911',$2,$3,4999,999.8,3999.2,'dummy','stripe',true,'paid',NOW())
		ON CONFLICT (public_txn_id) DO NOTHING
	`, payID, student1, courseID)
	_ = db.QueryRowContext(ctx, `SELECT id FROM payments WHERE public_txn_id='TXN-DUMMY-18402911'`).Scan(&payID)
	exec(ctx, db, `
		INSERT INTO enrollments (id, user_id, course_id, payment_id, payment_status, progress)
		VALUES ($1,$2,$3,$4,'paid',16.67)
		ON CONFLICT (user_id, course_id) DO UPDATE SET payment_status='paid', progress=16.67
	`, uuid.New(), student1, courseID, payID)
	exec(ctx, db, `
		INSERT INTO enrollments (id, user_id, course_id, payment_status, progress)
		VALUES ($1,$2,$3,'paid',45)
		ON CONFLICT (user_id, course_id) DO UPDATE SET payment_status='paid'
	`, uuid.New(), student2, courseID)
	exec(ctx, db, `
		INSERT INTO transactions (id, payment_id, course_id, course_title, creator_type, instructor_id, instructor_name, student_id, student_name, student_email, price, admin_commission_rate, admin_commission_amount, instructor_earnings, payment_method, status)
		SELECT $1,$2,$3,'Reactive Accelerator','instructor',$4,'Tapas Adhikary',$5,'Karim Rahman','student@nexurahub.com',3999.2,0.05,199.96,3799.24,'Dummy','completed'
		WHERE NOT EXISTS (SELECT 1 FROM transactions WHERE payment_id=$2)
	`, uuid.New(), payID, courseID, instructorID, student1)
	_ = adminID
}

func ensureReview(ctx context.Context, db *sql.DB, studentID, courseID uuid.UUID) {
	exec(ctx, db, `
		INSERT INTO reviews (id, course_id, user_id, rating, comment)
		VALUES ($1,$2,$3,5,'Outstanding walkthrough of React 19 and Go APIs.')
		ON CONFLICT (course_id, user_id) DO UPDATE SET rating=5, comment=EXCLUDED.comment
	`, uuid.New(), courseID, studentID)
}

func ensureLive(ctx context.Context, db *sql.DB, instructorID, courseID uuid.UUID) {
	exec(ctx, db, `
		INSERT INTO live_classes (id, instructor_id, course_id, title, description, date, time, duration, meeting_link)
		SELECT $1,$2,$3,'Live Career Roadmap Q&A','Ask anything about React, Go, and job prep.','15 Nov 2026','08:00 PM','90 min','https://meet.google.com/nex-ura-hub'
		WHERE NOT EXISTS (SELECT 1 FROM live_classes WHERE instructor_id=$2 AND title='Live Career Roadmap Q&A')
	`, uuid.New(), instructorID, courseID)
}

func ensureQuiz(ctx context.Context, db *sql.DB, instructorID uuid.UUID) {
	quizID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	exec(ctx, db, `
		INSERT INTO quiz_sets (id, instructor_id, title, description, total_marks, is_published)
		VALUES ($1,$2,'React 19 Fundamentals','Hooks, RSC, and forms.',5,true)
		ON CONFLICT (id) DO UPDATE SET instructor_id=EXCLUDED.instructor_id, is_published=true
	`, quizID, instructorID)
	qID := uuid.MustParse("22222222-3333-4444-5555-666666666666")
	exec(ctx, db, `
		INSERT INTO quiz_questions (id, quiz_set_id, title, description, points, position)
		VALUES ($1,$2,'Which hook is used for side effects?','Pick the correct React hook.',5,1)
		ON CONFLICT (id) DO NOTHING
	`, qID, quizID)
	exec(ctx, db, `INSERT INTO quiz_options (id, question_id, label, is_correct, position) VALUES ($1,$2,'useEffect',true,1) ON CONFLICT (id) DO NOTHING`,
		uuid.MustParse("33333333-4444-5555-6666-777777777777"), qID)
	exec(ctx, db, `INSERT INTO quiz_options (id, question_id, label, is_correct, position) VALUES ($1,$2,'useState',false,2) ON CONFLICT (id) DO NOTHING`,
		uuid.MustParse("33333333-4444-5555-6666-777777777778"), qID)
}

func ensureChat(ctx context.Context, db *sql.DB, courseID, instructorID, studentID uuid.UUID) {
	groupID := uuid.MustParse("cccccccc-dddd-eeee-ffff-000000000001")
	exec(ctx, db, `
		INSERT INTO conversations (id, type, name, course_id, instructor_id)
		VALUES ($1,'group','Reactive Accelerator Community Group',$2,$3)
		ON CONFLICT (id) DO UPDATE SET instructor_id=EXCLUDED.instructor_id, course_id=EXCLUDED.course_id
	`, groupID, courseID, instructorID)
	exec(ctx, db, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, groupID, instructorID)
	exec(ctx, db, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, groupID, studentID)
	exec(ctx, db, `INSERT INTO messages (id, conversation_id, sender_id, content) VALUES ($1,$2,$3,$4) ON CONFLICT (id) DO NOTHING`,
		uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-000000000002"), groupID, instructorID, "Welcome to the official group chat for Reactive Accelerator!")
}

func ensureNotifications(ctx context.Context, db *sql.DB, adminID, instructorID, student1, student2, courseID uuid.UUID) {
	_ = courseID
	exec(ctx, db, `
		INSERT INTO notifications (id, user_id, type, title, message, link)
		SELECT $1,$2,'enrollment','New student enrolled','Karim Rahman enrolled in Reactive Accelerator.','/dashboard/enrollments'
		WHERE NOT EXISTS (SELECT 1 FROM notifications WHERE user_id=$2 AND title='New student enrolled')
	`, uuid.New(), instructorID)
	exec(ctx, db, `
		INSERT INTO notifications (id, user_id, type, title, message, link)
		SELECT $1,$2,'payment','Dummy payment received','৳3999.20 credited split: 5% admin / 95% instructor.','/admin/wallet'
		WHERE NOT EXISTS (SELECT 1 FROM notifications WHERE user_id=$2 AND title='Dummy payment received')
	`, uuid.New(), adminID)
	exec(ctx, db, `
		INSERT INTO notifications (id, user_id, type, title, message, link)
		SELECT $1,$2,'enrollment','Enrolled successfully','You are enrolled in Reactive Accelerator.','/account/enrolled-courses'
		WHERE NOT EXISTS (SELECT 1 FROM notifications WHERE user_id=$2 AND title='Enrolled successfully')
	`, uuid.New(), student1)
	exec(ctx, db, `
		INSERT INTO notifications (id, user_id, type, title, message, link)
		SELECT $1,$2,'live','Live Class Starting Soon','Career Roadmap Q&A is on 15 Nov, 8:00 PM.','/dashboard/lives'
		WHERE NOT EXISTS (SELECT 1 FROM notifications WHERE user_id=$2 AND title='Live Class Starting Soon')
	`, uuid.New(), student2)
}

func exec(ctx context.Context, db *sql.DB, q string, args ...any) {
	if _, err := db.ExecContext(ctx, q, args...); err != nil {
		log.Printf("[seed warning] %v", err)
	}
}
