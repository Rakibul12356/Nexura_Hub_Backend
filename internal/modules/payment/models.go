package payment

import (
	"time"

	"github.com/google/uuid"
)

type DummyPayDTO struct {
	CourseID   string `json:"courseId" binding:"required"`
	CouponCode string `json:"couponCode"`
	Gateway    string `json:"gateway"`
}

type FreeEnrollDTO struct {
	CourseID string `json:"courseId" binding:"required"`
}

type CouponDTO struct {
	Code            string  `json:"code" binding:"required"`
	DiscountType    string  `json:"discountType" binding:"required"`
	DiscountValue   float64 `json:"discountValue" binding:"required"`
	ExpiryDate      string  `json:"expiryDate" binding:"required"`
	MaxRedemptions  int     `json:"maxRedemptions"`
	IsActive        *bool   `json:"isActive"`
}

type Coupon struct {
	ID               uuid.UUID `json:"id"`
	Code             string    `json:"code"`
	DiscountType     string    `json:"discountType"`
	DiscountValue    float64   `json:"discountValue"`
	ExpiryDate       string    `json:"expiryDate"`
	MaxRedemptions   int       `json:"maxRedemptions"`
	RedemptionCount  int       `json:"redemptionCount"`
	IsActive         bool      `json:"isActive"`
}

type ValidateDTO struct {
	Code     string `json:"code" binding:"required"`
	CourseID string `json:"courseId" binding:"required"`
}

type WalletLedger struct {
	ID           uuid.UUID `json:"id"`
	EntryType    string    `json:"entryType"`
	Amount       float64   `json:"amount"`
	BalanceAfter float64   `json:"balanceAfter"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"createdAt"`
}

type WalletView struct {
	OwnerType        string         `json:"ownerType"`
	Balance          float64        `json:"balance"`
	Currency         string         `json:"currency"`
	IsDummy          bool           `json:"isDummy"`
	LifetimeCredits  float64        `json:"lifetimeCredits"`
	Recent           []WalletLedger `json:"recent"`
}

type PlatformTransaction struct {
	ID                    string  `json:"id"`
	CourseID              string  `json:"courseId"`
	CourseTitle           string  `json:"courseTitle"`
	CreatorType           string  `json:"creatorType"`
	InstructorName        string  `json:"instructorName"`
	StudentName           string  `json:"studentName"`
	StudentEmail          string  `json:"studentEmail"`
	Price                 float64 `json:"price"`
	AdminCommissionRate   float64 `json:"adminCommissionRate"`
	AdminCommissionAmount float64 `json:"adminCommissionAmount"`
	InstructorEarnings    float64 `json:"instructorEarnings"`
	Date                  string  `json:"date"`
	Status                string  `json:"status"`
	PaymentMethod         string  `json:"paymentMethod"`
}

type StudentEnrollment struct {
	ID            uuid.UUID `json:"id"`
	StudentName   string    `json:"studentName"`
	StudentEmail  string    `json:"studentEmail"`
	StudentAvatar *string   `json:"studentAvatar"`
	CourseTitle   string    `json:"courseTitle"`
	EnrolledDate  string    `json:"enrolledDate"`
	Progress      float64   `json:"progress"`
	PaymentStatus string    `json:"paymentStatus"`
	CourseID      uuid.UUID `json:"courseId,omitempty"`
}
