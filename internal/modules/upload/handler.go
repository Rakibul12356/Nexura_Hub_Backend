package upload

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nexura-backend/internal/core/response"
)

type UploadHandler struct {
	dir string
}

func NewUploadHandler() *UploadHandler {
	_ = os.MkdirAll("uploads", 0o755)
	return &UploadHandler{dir: "uploads"}
}

func (h *UploadHandler) UploadImage(c *gin.Context) { h.save(c, 5<<20, []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}) }
func (h *UploadHandler) UploadPDF(c *gin.Context)   { h.save(c, 20<<20, []string{".pdf"}) }
func (h *UploadHandler) UploadVideo(c *gin.Context) { h.save(c, 500<<20, []string{".mp4", ".webm", ".mov", ".m3u8"}) }
func (h *UploadHandler) UploadFile(c *gin.Context)  { h.save(c, 50<<20, nil) }
func (h *UploadHandler) UploadAvatar(c *gin.Context) { h.save(c, 5<<20, []string{".jpg", ".jpeg", ".png", ".webp"}) }

func (h *UploadHandler) save(c *gin.Context, max int64, exts []string) {
	file, err := c.FormFile("file")
	if err != nil {
		response.BindError(c, err)
		return
	}
	if file.Size > max {
		response.Error(c, http.StatusBadRequest, "File too large", "VALIDATION_ERROR", nil)
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if len(exts) > 0 {
		ok := false
		for _, e := range exts {
			if ext == e {
				ok = true
				break
			}
		}
		if !ok {
			response.Error(c, http.StatusBadRequest, "Unsupported file type", "VALIDATION_ERROR", nil)
			return
		}
	}
	name := uuid.New().String() + ext
	dst := filepath.Join(h.dir, name)
	src, err := file.Open()
	if err != nil {
		response.Internal(c, err)
		return
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		response.Internal(c, err)
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, src); err != nil {
		response.Internal(c, err)
		return
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/uploads/%s", scheme, c.Request.Host, name)
	response.Success(c, http.StatusOK, "Uploaded", gin.H{"url": url, "size": formatSize(file.Size)})
}

func (h *UploadHandler) DeleteFile(c *gin.Context) {
	var dto struct {
		FileURL string `json:"fileUrl"`
	}
	_ = c.ShouldBindJSON(&dto)
	response.Success(c, http.StatusOK, "File deleted successfully", gin.H{})
}

func formatSize(n int64) string {
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
}
