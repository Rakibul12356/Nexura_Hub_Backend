package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	appErrors "nexura-backend/internal/core/errors"
	"nexura-backend/pkg/utils"
)

type Repository interface {
	DummyPay(ctx context.Context, userID uuid.UUID, courseRef, couponCode, gatewayLabel string) (map[string]any, error)
	FreeEnroll(ctx context.Context, userID uuid.UUID, courseRef string) (map[string]any, error)
	GetPayment(ctx context.Context, id string, userID uuid.UUID, isAdmin bool) (map[string]any, error)
	ListCoupons(ctx context.Context) ([]Coupon, error)
	CreateCoupon(ctx context.Context, c *Coupon, createdBy uuid.UUID) error
	PatchCoupon(ctx context.Context, id uuid.UUID, active *bool) (*Coupon, error)
	DeleteCoupon(ctx context.Context, id uuid.UUID) error
	ValidateCoupon(ctx context.Context, code, courseRef string) (map[string]any, error)
	GetWallet(ctx context.Context, ownerType string, userID uuid.UUID) (*WalletView, error)
	EnrollmentStatus(ctx context.Context, userID uuid.UUID, courseRef string) (map[string]any, error)
	ListCourseEnrollments(ctx context.Context, courseID uuid.UUID, search string) ([]StudentEnrollment, error)
	ListInstructorEnrollments(ctx context.Context, instructorID uuid.UUID, limit int) ([]StudentEnrollment, error)
	CompleteLesson(ctx context.Context, userID, lessonID uuid.UUID) (map[string]any, error)
	EnsureAdminWallet(ctx context.Context) error
}

type postgresRepo struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return &postgresRepo{db: db} }

func (r *postgresRepo) DummyPay(ctx context.Context, userID uuid.UUID, courseRef, couponCode, gatewayLabel string) (map[string]any, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var courseID uuid.UUID
	var title, creatorType string
	var instructorID uuid.UUID
	var price, discountPrice sql.NullFloat64
	var published bool
	err = tx.QueryRowContext(ctx, `
		SELECT id, title, creator_type, instructor_id, price, discount_price, is_published
		FROM courses WHERE deleted_at IS NULL AND (id::text=$1 OR slug=$1) FOR UPDATE
	`, courseRef).Scan(&courseID, &title, &creatorType, &instructorID, &price, &discountPrice, &published)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrCourseNotFound
	}
	if err != nil {
		return nil, err
	}
	if !published {
		return nil, appErrors.ErrCourseNotPublished
	}

	var enrolled bool
	_ = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM enrollments WHERE user_id=$1 AND course_id=$2)`, userID, courseID).Scan(&enrolled)
	if enrolled {
		return nil, appErrors.ErrAlreadyEnrolled
	}

	original := price.Float64
	if discountPrice.Valid && discountPrice.Float64 > 0 && discountPrice.Float64 < original {
		original = discountPrice.Float64
	}

	var couponID *uuid.UUID
	discountAmt := 0.0
	if couponCode != "" {
		cid, dtype, dval, err := r.lockCoupon(ctx, tx, couponCode)
		if err != nil {
			return nil, err
		}
		couponID = &cid
		if dtype == "percentage" {
			discountAmt = utils.RoundMoney(original * dval / 100)
		} else {
			discountAmt = dval
		}
		if discountAmt > original {
			discountAmt = original
		}
	}
	amountPaid := utils.RoundMoney(original - discountAmt)
	if amountPaid < 0 {
		amountPaid = 0
	}

	adminRate := 1.0
	if creatorType == "instructor" {
		adminRate = 0.05
	}
	adminCut := utils.RoundMoney(amountPaid * adminRate)
	instrEarn := utils.RoundMoney(amountPaid - adminCut)
	if creatorType != "instructor" {
		instrEarn = 0
		adminCut = amountPaid
		adminRate = 1
	}

	txnPublic := utils.DummyTxnID()
	paymentID := uuid.New()
	label := gatewayLabel
	if label == "" {
		label = "dummy"
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payments (id, public_txn_id, user_id, course_id, coupon_id, original_price, discount_amount, amount_paid, gateway, gateway_label, is_dummy, status, paid_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'dummy',$9,true,'paid',NOW())
	`, paymentID, txnPublic, userID, courseID, couponID, original, discountAmt, amountPaid, label)
	if err != nil {
		return nil, err
	}

	enrollID := uuid.New()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO enrollments (id, user_id, course_id, payment_id, payment_status, progress)
		VALUES ($1,$2,$3,$4,'paid',0)
	`, enrollID, userID, courseID, paymentID)
	if err != nil {
		return nil, err
	}

	var first, last, email, instrFirst, instrLast string
	_ = tx.QueryRowContext(ctx, `SELECT first_name, last_name, email FROM users WHERE id=$1`, userID).Scan(&first, &last, &email)
	_ = tx.QueryRowContext(ctx, `SELECT first_name, last_name FROM users WHERE id=$1`, instructorID).Scan(&instrFirst, &instrLast)
	studentName := strings.TrimSpace(first + " " + last)
	instrName := strings.TrimSpace(instrFirst + " " + instrLast)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (id, payment_id, course_id, course_title, creator_type, instructor_id, instructor_name, student_id, student_name, student_email, price, admin_commission_rate, admin_commission_amount, instructor_earnings, payment_method, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'Dummy','completed')
	`, uuid.New(), paymentID, courseID, title, creatorType, instructorID, instrName, userID, studentName, email, amountPaid, adminRate, adminCut, instrEarn)
	if err != nil {
		return nil, err
	}

	adminBal := 0.0
	if amountPaid > 0 {
		adminBal, err = r.creditWallet(ctx, tx, "admin", nil, paymentID, adminCut, fmt.Sprintf("%.0f%% dummy commission · %s", adminRate*100, txnPublic))
		if err != nil {
			return nil, err
		}
		if creatorType == "instructor" && instrEarn > 0 {
			if _, err := r.creditWallet(ctx, tx, "instructor", &instructorID, paymentID, instrEarn, "95% dummy earnings · "+txnPublic); err != nil {
				return nil, err
			}
		}
	}

	if couponID != nil {
		_, _ = tx.ExecContext(ctx, `UPDATE coupons SET redemption_count = redemption_count+1 WHERE id=$1`, *couponID)
		_, _ = tx.ExecContext(ctx, `INSERT INTO coupon_redemptions (id, coupon_id, user_id, course_id, payment_id) VALUES ($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`,
			uuid.New(), *couponID, userID, courseID, paymentID)
	}

	if err := r.autoJoinChat(ctx, tx, userID, instructorID, courseID, title, studentName); err != nil {
		return nil, err
	}

	_, _ = tx.ExecContext(ctx, `INSERT INTO notifications (id, user_id, type, title, message, link) VALUES ($1,$2,'enrollment','New enrollment', $3, $4)`,
		uuid.New(), instructorID, studentName+" enrolled in "+title, "/dashboard/courses/"+courseID.String()+"/enrollments")
	var adminID uuid.UUID
	_ = tx.QueryRowContext(ctx, `SELECT id FROM users WHERE role='admin' AND deleted_at IS NULL ORDER BY created_at LIMIT 1`).Scan(&adminID)
	if adminID != uuid.Nil {
		_, _ = tx.ExecContext(ctx, `INSERT INTO notifications (id, user_id, type, title, message, link) VALUES ($1,$2,'payment','Dummy payment', $3, '/admin/revenue')`,
			uuid.New(), adminID, txnPublic+" · "+title+" · ৳"+fmt.Sprintf("%.2f", amountPaid))
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return map[string]any{
		"paymentId":      paymentID,
		"transactionId":  txnPublic,
		"isDummy":        true,
		"courseId":       courseID,
		"courseTitle":    title,
		"originalPrice":  original,
		"discountAmount": discountAmt,
		"amountPaid":     amountPaid,
		"gateway":        "dummy",
		"gatewayLabel":   label,
		"paymentMethod":  "Dummy",
		"paidAt":         time.Now().UTC().Format(time.RFC3339),
		"enrollment": map[string]any{
			"id": enrollID, "isEnrolled": true, "paymentStatus": "paid",
		},
		"split": map[string]any{
			"creatorType": creatorType, "adminCommissionRate": adminRate,
			"adminCommissionAmount": adminCut, "instructorEarnings": instrEarn,
		},
		"adminWallet": map[string]any{"credited": adminCut, "balanceAfter": adminBal},
	}, nil
}

func (r *postgresRepo) lockCoupon(ctx context.Context, tx *sql.Tx, code string) (uuid.UUID, string, float64, error) {
	var id uuid.UUID
	var dtype string
	var dval float64
	var expiry time.Time
	var maxR, count int
	var active bool
	err := tx.QueryRowContext(ctx, `
		SELECT id, discount_type, discount_value, expiry_date, max_redemptions, redemption_count, is_active
		FROM coupons WHERE UPPER(code)=UPPER($1) FOR UPDATE
	`, code).Scan(&id, &dtype, &dval, &expiry, &maxR, &count, &active)
	if err != nil {
		return uuid.Nil, "", 0, appErrors.ErrCouponInvalid
	}
	if !active || time.Now().After(expiry.Add(24*time.Hour)) || count >= maxR {
		return uuid.Nil, "", 0, appErrors.ErrCouponInvalid
	}
	return id, dtype, dval, nil
}

func (r *postgresRepo) creditWallet(ctx context.Context, tx *sql.Tx, ownerType string, userID *uuid.UUID, paymentID uuid.UUID, amount float64, note string) (float64, error) {
	if amount <= 0 {
		return 0, nil
	}
	var uid uuid.UUID
	if ownerType == "admin" {
		err := tx.QueryRowContext(ctx, `SELECT id FROM users WHERE role='admin' AND deleted_at IS NULL ORDER BY created_at LIMIT 1`).Scan(&uid)
		if err != nil {
			return 0, err
		}
	} else if userID != nil {
		uid = *userID
	}
	_, _ = tx.ExecContext(ctx, `INSERT INTO wallets (id, owner_type, user_id, balance) VALUES ($1,$2,$3,0) ON CONFLICT (owner_type, user_id) DO NOTHING`, uuid.New(), ownerType, uid)
	var walletID uuid.UUID
	var bal float64
	if err := tx.QueryRowContext(ctx, `SELECT id, balance FROM wallets WHERE owner_type=$1 AND user_id=$2 FOR UPDATE`, ownerType, uid).Scan(&walletID, &bal); err != nil {
		return 0, err
	}
	newBal := utils.RoundMoney(bal + amount)
	_, err := tx.ExecContext(ctx, `UPDATE wallets SET balance=$1, updated_at=NOW() WHERE id=$2`, newBal, walletID)
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO wallet_ledger (id, wallet_id, payment_id, entry_type, amount, balance_after, note) VALUES ($1,$2,$3,'credit',$4,$5,$6)`,
		uuid.New(), walletID, paymentID, amount, newBal, note)
	return newBal, err
}

func (r *postgresRepo) autoJoinChat(ctx context.Context, tx *sql.Tx, studentID, instructorID, courseID uuid.UUID, courseTitle, studentName string) error {
	var groupID uuid.UUID
	err := tx.QueryRowContext(ctx, `SELECT id FROM conversations WHERE course_id=$1 AND type='group'`, courseID).Scan(&groupID)
	if errors.Is(err, sql.ErrNoRows) {
		groupID = uuid.New()
		_, err = tx.ExecContext(ctx, `INSERT INTO conversations (id, type, name, course_id, instructor_id) VALUES ($1,'group',$2,$3,$4)`,
			groupID, courseTitle+" Community Group", courseID, instructorID)
		if err != nil {
			return err
		}
		_, _ = tx.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, groupID, instructorID)
		_, _ = tx.ExecContext(ctx, `INSERT INTO messages (id, conversation_id, sender_id, content) VALUES ($1,$2,$3,$4)`,
			uuid.New(), groupID, instructorID, "Welcome to the official group chat for "+courseTitle+"!")
	}
	_, _ = tx.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, groupID, studentID)
	_, _ = tx.ExecContext(ctx, `INSERT INTO messages (id, conversation_id, sender_id, content) VALUES ($1,$2,$3,$4)`,
		uuid.New(), groupID, instructorID, "Welcome "+studentName+" to the "+courseTitle+" community group!")

	a, b := studentID, instructorID
	if b.String() < a.String() {
		a, b = b, a
	}
	var dmID uuid.UUID
	err = tx.QueryRowContext(ctx, `SELECT conversation_id FROM direct_pairs WHERE user_a=$1 AND user_b=$2`, a, b).Scan(&dmID)
	if errors.Is(err, sql.ErrNoRows) {
		dmID = uuid.New()
		_, err = tx.ExecContext(ctx, `INSERT INTO conversations (id, type, name, course_id, instructor_id) VALUES ($1,'direct',$2,$3,$4)`,
			dmID, "Direct", courseID, instructorID)
		if err != nil {
			return err
		}
		_, _ = tx.ExecContext(ctx, `INSERT INTO direct_pairs (conversation_id, user_a, user_b) VALUES ($1,$2,$3)`, dmID, a, b)
		_, _ = tx.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, dmID, studentID)
		_, _ = tx.ExecContext(ctx, `INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, dmID, instructorID)
		_, _ = tx.ExecContext(ctx, `INSERT INTO messages (id, conversation_id, sender_id, content) VALUES ($1,$2,$3,$4)`,
			uuid.New(), dmID, instructorID, "Welcome to "+courseTitle+"! Feel free to send me any direct questions.")
	}
	return nil
}

func (r *postgresRepo) FreeEnroll(ctx context.Context, userID uuid.UUID, courseRef string) (map[string]any, error) {
	var courseID uuid.UUID
	var price float64
	var published bool
	err := r.db.QueryRowContext(ctx, `SELECT id, price, is_published FROM courses WHERE deleted_at IS NULL AND (id::text=$1 OR slug=$1)`, courseRef).
		Scan(&courseID, &price, &published)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrCourseNotFound
	}
	if err != nil {
		return nil, err
	}
	if !published {
		return nil, appErrors.ErrCourseNotPublished
	}
	if price > 0 {
		return nil, appErrors.ErrFreeEnrollOnly
	}
	return r.DummyPay(ctx, userID, courseRef, "", "dummy")
}

func (r *postgresRepo) GetPayment(ctx context.Context, id string, userID uuid.UUID, isAdmin bool) (map[string]any, error) {
	q := `
		SELECT p.id, p.public_txn_id, p.user_id, p.course_id, c.title, p.original_price, p.discount_amount, p.amount_paid,
		       p.gateway, p.gateway_label, p.is_dummy, p.status, p.paid_at
		FROM payments p JOIN courses c ON c.id=p.course_id
		WHERE p.id::text=$1 OR p.public_txn_id=$1
	`
	var pid, uid, cid uuid.UUID
	var txn, title, gateway, label, status string
	var orig, disc, paid float64
	var dummy bool
	var paidAt time.Time
	err := r.db.QueryRowContext(ctx, q, id).Scan(&pid, &txn, &uid, &cid, &title, &orig, &disc, &paid, &gateway, &label, &dummy, &status, &paidAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}
	if !isAdmin && uid != userID {
		return nil, appErrors.ErrForbidden
	}
	return map[string]any{
		"paymentId": pid, "transactionId": txn, "courseId": cid, "courseTitle": title,
		"originalPrice": orig, "discountAmount": disc, "amountPaid": paid,
		"gateway": gateway, "gatewayLabel": label, "isDummy": dummy, "status": status,
		"paymentMethod": "Dummy", "paidAt": paidAt.UTC().Format(time.RFC3339),
	}, nil
}

func (r *postgresRepo) ListCoupons(ctx context.Context) ([]Coupon, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, code, discount_type, discount_value, expiry_date, max_redemptions, redemption_count, is_active FROM coupons ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Coupon
	for rows.Next() {
		var c Coupon
		var exp time.Time
		if err := rows.Scan(&c.ID, &c.Code, &c.DiscountType, &c.DiscountValue, &exp, &c.MaxRedemptions, &c.RedemptionCount, &c.IsActive); err != nil {
			return nil, err
		}
		c.ExpiryDate = exp.Format("2006-01-02")
		out = append(out, c)
	}
	if out == nil {
		out = []Coupon{}
	}
	return out, nil
}

func (r *postgresRepo) CreateCoupon(ctx context.Context, c *Coupon, createdBy uuid.UUID) error {
	c.ID = uuid.New()
	c.Code = strings.ToUpper(strings.TrimSpace(c.Code))
	if c.MaxRedemptions == 0 {
		c.MaxRedemptions = 200
	}
	exp, err := time.Parse("2006-01-02", c.ExpiryDate)
	if err != nil {
		return err
	}
	c.IsActive = true
	return r.db.QueryRowContext(ctx, `
		INSERT INTO coupons (id, code, discount_type, discount_value, expiry_date, max_redemptions, is_active, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,true,$7) RETURNING is_active
	`, c.ID, c.Code, c.DiscountType, c.DiscountValue, exp, c.MaxRedemptions, createdBy).Scan(&c.IsActive)
}

func (r *postgresRepo) PatchCoupon(ctx context.Context, id uuid.UUID, active *bool) (*Coupon, error) {
	if active != nil {
		_, err := r.db.ExecContext(ctx, `UPDATE coupons SET is_active=$1 WHERE id=$2`, *active, id)
		if err != nil {
			return nil, err
		}
	}
	var c Coupon
	var exp time.Time
	err := r.db.QueryRowContext(ctx, `SELECT id, code, discount_type, discount_value, expiry_date, max_redemptions, redemption_count, is_active FROM coupons WHERE id=$1`, id).
		Scan(&c.ID, &c.Code, &c.DiscountType, &c.DiscountValue, &exp, &c.MaxRedemptions, &c.RedemptionCount, &c.IsActive)
	c.ExpiryDate = exp.Format("2006-01-02")
	return &c, err
}

func (r *postgresRepo) DeleteCoupon(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM coupons WHERE id=$1`, id)
	return err
}

func (r *postgresRepo) ValidateCoupon(ctx context.Context, code, courseRef string) (map[string]any, error) {
	var price float64
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(discount_price, price) FROM courses WHERE deleted_at IS NULL AND (id::text=$1 OR slug=$1)`, courseRef).Scan(&price)
	if err != nil {
		return nil, appErrors.ErrCourseNotFound
	}
	var dtype string
	var dval float64
	var expiry time.Time
	var maxR, count int
	var active bool
	err = r.db.QueryRowContext(ctx, `SELECT discount_type, discount_value, expiry_date, max_redemptions, redemption_count, is_active FROM coupons WHERE UPPER(code)=UPPER($1)`, code).
		Scan(&dtype, &dval, &expiry, &maxR, &count, &active)
	if err != nil || !active || time.Now().After(expiry.Add(24*time.Hour)) || count >= maxR {
		return nil, appErrors.ErrCouponInvalid
	}
	disc := dval
	if dtype == "percentage" {
		disc = utils.RoundMoney(price * dval / 100)
	}
	if disc > price {
		disc = price
	}
	return map[string]any{
		"valid": true, "code": strings.ToUpper(code), "discountType": dtype, "discountValue": dval,
		"discountAmount": disc, "finalPrice": utils.RoundMoney(price - disc),
	}, nil
}

func (r *postgresRepo) GetWallet(ctx context.Context, ownerType string, userID uuid.UUID) (*WalletView, error) {
	empty := &WalletView{OwnerType: ownerType, Balance: 0, Currency: "BDT", IsDummy: true, LifetimeCredits: 0, Recent: []WalletLedger{}}
	uid := userID
	if ownerType == "admin" {
		if err := r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE role='admin' AND deleted_at IS NULL ORDER BY created_at LIMIT 1`).Scan(&uid); err != nil {
			return empty, nil
		}
	}
	if uid == uuid.Nil {
		return empty, nil
	}
	_, _ = r.db.ExecContext(ctx, `
		INSERT INTO wallets (id, owner_type, user_id, balance)
		SELECT $1, $2, $3, 0
		WHERE NOT EXISTS (
			SELECT 1 FROM wallets WHERE owner_type::text = $2 AND user_id = $3
		)
	`, uuid.New(), ownerType, uid)
	var walletID uuid.UUID
	var bal float64
	err := r.db.QueryRowContext(ctx, `SELECT id, balance FROM wallets WHERE owner_type::text=$1 AND user_id=$2`, ownerType, uid).Scan(&walletID, &bal)
	if err != nil {
		return empty, nil
	}
	var life float64
	_ = r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount),0) FROM wallet_ledger WHERE wallet_id=$1 AND entry_type::text='credit'`, walletID).Scan(&life)
	rows, err := r.db.QueryContext(ctx, `SELECT id, entry_type::text, amount, balance_after, COALESCE(note,''), created_at FROM wallet_ledger WHERE wallet_id=$1 ORDER BY created_at DESC LIMIT 20`, walletID)
	if err != nil {
		return &WalletView{OwnerType: ownerType, Balance: bal, Currency: "BDT", IsDummy: true, LifetimeCredits: life, Recent: []WalletLedger{}}, nil
	}
	defer rows.Close()
	recent := []WalletLedger{}
	for rows.Next() {
		var l WalletLedger
		if err := rows.Scan(&l.ID, &l.EntryType, &l.Amount, &l.BalanceAfter, &l.Note, &l.CreatedAt); err != nil {
			continue
		}
		recent = append(recent, l)
	}
	return &WalletView{OwnerType: ownerType, Balance: bal, Currency: "BDT", IsDummy: true, LifetimeCredits: life, Recent: recent}, nil
}

func (r *postgresRepo) EnrollmentStatus(ctx context.Context, userID uuid.UUID, courseRef string) (map[string]any, error) {
	var courseID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT id FROM courses WHERE deleted_at IS NULL AND (id::text=$1 OR slug=$1)`, courseRef).Scan(&courseID)
	if err != nil {
		return nil, appErrors.ErrCourseNotFound
	}
	var eID uuid.UUID
	var progress float64
	var status string
	err = r.db.QueryRowContext(ctx, `SELECT id, progress, payment_status FROM enrollments WHERE user_id=$1 AND course_id=$2`, userID, courseID).Scan(&eID, &progress, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return map[string]any{"isEnrolled": false}, nil
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{"isEnrolled": true, "id": eID, "progress": progress, "paymentStatus": status}, nil
}

func (r *postgresRepo) ListCourseEnrollments(ctx context.Context, courseID uuid.UUID, search string) ([]StudentEnrollment, error) {
	q := `
		SELECT e.id, u.first_name || ' ' || u.last_name, u.email, u.avatar, c.title, e.enrolled_at, e.progress, e.payment_status, e.course_id
		FROM enrollments e JOIN users u ON u.id=e.user_id JOIN courses c ON c.id=e.course_id
		WHERE e.course_id=$1
	`
	args := []any{courseID}
	if search != "" {
		q += ` AND (u.first_name ILIKE $2 OR u.last_name ILIKE $2 OR u.email ILIKE $2)`
		args = append(args, "%"+search+"%")
	}
	q += ` ORDER BY e.enrolled_at DESC`
	return r.scanEnrollments(ctx, q, args...)
}

func (r *postgresRepo) ListInstructorEnrollments(ctx context.Context, instructorID uuid.UUID, limit int) ([]StudentEnrollment, error) {
	if limit < 1 {
		limit = 50
	}
	q := `
		SELECT e.id, u.first_name || ' ' || u.last_name, u.email, u.avatar, c.title, e.enrolled_at, e.progress, e.payment_status, e.course_id
		FROM enrollments e JOIN users u ON u.id=e.user_id JOIN courses c ON c.id=e.course_id
		WHERE c.instructor_id=$1 ORDER BY e.enrolled_at DESC LIMIT $2
	`
	return r.scanEnrollments(ctx, q, instructorID, limit)
}

func (r *postgresRepo) scanEnrollments(ctx context.Context, q string, args ...any) ([]StudentEnrollment, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StudentEnrollment
	for rows.Next() {
		var e StudentEnrollment
		var at time.Time
		if err := rows.Scan(&e.ID, &e.StudentName, &e.StudentEmail, &e.StudentAvatar, &e.CourseTitle, &at, &e.Progress, &e.PaymentStatus, &e.CourseID); err != nil {
			return nil, err
		}
		e.EnrolledDate = at.Format("2006-01-02")
		out = append(out, e)
	}
	if out == nil {
		out = []StudentEnrollment{}
	}
	return out, nil
}

func (r *postgresRepo) CompleteLesson(ctx context.Context, userID, lessonID uuid.UUID) (map[string]any, error) {
	var courseID uuid.UUID
	err := r.db.QueryRowContext(ctx, `SELECT m.course_id FROM lessons l JOIN modules m ON m.id=l.module_id WHERE l.id=$1`, lessonID).Scan(&courseID)
	if err != nil {
		return nil, appErrors.ErrLessonNotFound
	}
	var enrolled bool
	_ = r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM enrollments WHERE user_id=$1 AND course_id=$2)`, userID, courseID).Scan(&enrolled)
	if !enrolled {
		return nil, appErrors.ErrNotEnrolled
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO lesson_progress (id, user_id, lesson_id, course_id, completed, completed_at)
		VALUES ($1,$2,$3,$4,true,NOW())
		ON CONFLICT (user_id, lesson_id) DO UPDATE SET completed=true, completed_at=NOW()
	`, uuid.New(), userID, lessonID, courseID)
	if err != nil {
		return nil, err
	}
	var total, done int
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM lessons l JOIN modules m ON m.id=l.module_id WHERE m.course_id=$1 AND l.is_published=true`, courseID).Scan(&total)
	_ = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM lesson_progress lp JOIN lessons l ON l.id=lp.lesson_id JOIN modules m ON m.id=l.module_id
		WHERE lp.user_id=$1 AND lp.course_id=$2 AND lp.completed=true AND l.is_published=true
	`, userID, courseID).Scan(&done)
	progress := 0.0
	if total > 0 {
		progress = utils.RoundMoney(float64(done) / float64(total) * 100)
	}
	if progress >= 100 {
		_, _ = r.db.ExecContext(ctx, `UPDATE enrollments SET progress=$1, completed_at=NOW() WHERE user_id=$2 AND course_id=$3`, progress, userID, courseID)
	} else {
		_, _ = r.db.ExecContext(ctx, `UPDATE enrollments SET progress=$1 WHERE user_id=$2 AND course_id=$3`, progress, userID, courseID)
	}
	return map[string]any{"completed": true, "progress": progress}, nil
}

func (r *postgresRepo) EnsureAdminWallet(ctx context.Context) error {
	var adminID uuid.UUID
	if err := r.db.QueryRowContext(ctx, `SELECT id FROM users WHERE role='admin' AND deleted_at IS NULL ORDER BY created_at LIMIT 1`).Scan(&adminID); err != nil {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO wallets (id, owner_type, user_id, balance) VALUES ($1,'admin',$2,0) ON CONFLICT (owner_type, user_id) DO NOTHING`, uuid.New(), adminID)
	return err
}
