package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type ErrorBody struct {
	Code    string      `json:"code"`
	Details interface{} `json:"details,omitempty"`
}

func Success(c *gin.Context, status int, message string, data interface{}) {
	if message == "" {
		message = "OK"
	}
	if data == nil {
		data = gin.H{}
	}
	c.JSON(status, gin.H{
		"success":    true,
		"statusCode": status,
		"message":    message,
		"data":       data,
	})
}

func SuccessWithMeta(c *gin.Context, status int, message string, data interface{}, meta Meta) {
	if message == "" {
		message = "OK"
	}
	if data == nil {
		data = []interface{}{}
	}
	c.JSON(status, gin.H{
		"success":    true,
		"statusCode": status,
		"message":    message,
		"data":       data,
		"meta":       meta,
	})
}

func Error(c *gin.Context, status int, message, code string, details interface{}) {
	if message == "" {
		message = http.StatusText(status)
	}
	if code == "" {
		code = http.StatusText(status)
	}
	body := gin.H{
		"success":    false,
		"statusCode": status,
		"message":    message,
		"error": ErrorBody{
			Code:    code,
			Details: details,
		},
	}
	c.JSON(status, body)
}

func BindError(c *gin.Context, err error) {
	Error(c, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR", nil)
}

func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = "Unauthorized"
	}
	Error(c, http.StatusUnauthorized, message, "UNAUTHORIZED", nil)
}

func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = "Forbidden"
	}
	Error(c, http.StatusForbidden, message, "FORBIDDEN", nil)
}

func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = "Not found"
	}
	Error(c, http.StatusNotFound, message, "NOT_FOUND", nil)
}

func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, message, "CONFLICT", nil)
}

func Unprocessable(c *gin.Context, message string) {
	Error(c, http.StatusUnprocessableEntity, message, "BUSINESS_RULE", nil)
}

func Internal(c *gin.Context, err error) {
	msg := "Internal server error"
	if err != nil {
		msg = err.Error()
	}
	Error(c, http.StatusInternalServerError, msg, "INTERNAL_ERROR", nil)
}
