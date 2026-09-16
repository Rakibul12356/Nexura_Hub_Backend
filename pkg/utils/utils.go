package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"nexura-backend/internal/core/response"
)

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func Slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = nonSlug.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "item"
	}
	return fmt.Sprintf("%s-%s", s, uuid.New().String()[:8])
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func RandomDigits(n int) string {
	const digits = "0123456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = digits[int(b[i])%10]
	}
	return string(out)
}

func RandomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func DummyTxnID() string {
	return "TXN-DUMMY-" + RandomDigits(8)
}

func CertificatePublicID() string {
	return "NEX-" + RandomDigits(6)
}

func PageMeta(page, limit, total int) response.Meta {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	pages := int(math.Ceil(float64(total) / float64(limit)))
	if pages < 1 {
		pages = 0
	}
	return response.Meta{Page: page, Limit: limit, Total: total, TotalPages: pages}
}

func Offset(page, limit int) int {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	return (page - 1) * limit
}

func Ptr[T any](v T) *T { return &v }

func DisplayTime(t time.Time) string {
	return t.UTC().Format("03:04 PM")
}

func RelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

func FullName(first, last string) string {
	return strings.TrimSpace(first + " " + last)
}

func RoundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
