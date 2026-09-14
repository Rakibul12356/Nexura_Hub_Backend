// internal/core/errors/errors.go
package errors

import "errors"

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrUserAlreadyExists     = errors.New("user with this email already exists")
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrUserSuspended         = errors.New("user account is suspended")
	ErrUnauthorized          = errors.New("unauthorized access")
	ErrForbidden             = errors.New("access forbidden")
	ErrAdminChatBlocked      = errors.New("admin users are restricted from accessing chat features")
	ErrCourseNotFound        = errors.New("course not found or not available")
	ErrCourseNotPublished    = errors.New("course is not published")
	ErrAlreadyEnrolled       = errors.New("student is already enrolled in this course")
	ErrConversationNotFound = errors.New("chat conversation not found")
	ErrInternalServer        = errors.New("internal server error")
)
