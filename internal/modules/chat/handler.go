// internal/modules/chat/handler.go
package chat

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandler struct {
	chatUsecase ChatUsecase
}

func NewChatHandler(chatUsecase ChatUsecase) *ChatHandler {
	return &ChatHandler{chatUsecase: chatUsecase}
}

func (h *ChatHandler) GetConversations(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(uuid.UUID)

	conversations, err := h.chatUsecase.GetUserConversations(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   conversations,
	})
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
	convIDStr := c.Param("id")
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid conversation ID"})
		return
	}

	cursorStr := c.Query("cursor")
	var cursorID *uuid.UUID
	if cursorStr != "" {
		if cid, err := uuid.Parse(cursorStr); err == nil {
			cursorID = &cid
		}
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	messages, err := h.chatUsecase.GetMessages(c.Request.Context(), convID, cursorID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	var nextCursor *uuid.UUID
	if len(messages) == limit {
		lastMsgID := messages[len(messages)-1].ID
		nextCursor = &lastMsgID
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"meta": gin.H{
			"limit":      limit,
			"nextCursor": nextCursor,
		},
		"data": messages,
	})
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(uuid.UUID)

	var dto SendMessageDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	msg, err := h.chatUsecase.SendMessage(c.Request.Context(), userID, dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   msg,
	})
}

func (h *ChatHandler) SendReaction(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	userID := userIDVal.(uuid.UUID)

	var dto SendReactionDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": err.Error()})
		return
	}

	reactions, err := h.chatUsecase.ToggleReaction(c.Request.Context(), userID, dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   reactions,
	})
}
