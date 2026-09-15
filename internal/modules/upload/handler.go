// internal/modules/upload/handler.go
package upload

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

func (h *UploadHandler) UploadImage(c *gin.Context) {
	fileId := uuid.New().String()
	// In production, save file to S3/Cloudinary/Local disk. Return sample URL.
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"fileId": fileId,
			"url":    "https://images.unsplash.com/photo-1516321318423-f06f85e504b3?w=800",
		},
	})
}

func (h *UploadHandler) UploadVideo(c *gin.Context) {
	fileId := uuid.New().String()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"fileId": fileId,
			"url":    "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4",
		},
	})
}

func (h *UploadHandler) UploadPDF(c *gin.Context) {
	fileId := uuid.New().String()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"fileId": fileId,
			"url":    "https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf",
		},
	})
}

func (h *UploadHandler) DeleteFile(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "File deleted successfully",
	})
}
