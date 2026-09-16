package chat

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"nexura-backend/internal/core/middleware"
	"nexura-backend/internal/core/response"
	"nexura-backend/pkg/utils"
)

type ChatHandler struct {
	chatUsecase ChatUsecase
}

func NewChatHandler(chatUsecase ChatUsecase) *ChatHandler {
	return &ChatHandler{chatUsecase: chatUsecase}
}

func (h *ChatHandler) GetConversations(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	list, err := h.chatUsecase.GetUserConversations(c.Request.Context(), userID)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", list)
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	conv, err := h.chatUsecase.GetConversation(c.Request.Context(), id, userID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "OK", conv)
}

func (h *ChatHandler) GetMessages(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	msgs, total, err := h.chatUsecase.GetMessages(c.Request.Context(), id, page, limit)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "OK", msgs, utils.PageMeta(page, limit, total))
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var dto SendMessageDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	msg, err := h.chatUsecase.SendMessage(c.Request.Context(), userID, dto)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Message sent", msg)
}

func (h *ChatHandler) SendReaction(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var dto SendReactionDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	if id := c.Param("id"); id != "" && dto.MessageID == uuid.Nil {
		dto.MessageID, _ = uuid.Parse(id)
	}
	reactions, err := h.chatUsecase.ToggleReaction(c.Request.Context(), userID, dto)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", reactions)
}

func (h *ChatHandler) CreateDirect(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var dto struct {
		UserID uuid.UUID `json:"userId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	conv, err := h.chatUsecase.GetOrCreateDirect(c.Request.Context(), userID, dto.UserID)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "OK", conv)
}

func (h *ChatHandler) CreateGroup(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var dto struct {
		CourseID  uuid.UUID `json:"courseId" binding:"required"`
		GroupName string    `json:"groupName"`
	}
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BindError(c, err)
		return
	}
	conv, err := h.chatUsecase.CreateGroup(c.Request.Context(), userID, dto.CourseID, dto.GroupName)
	if err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Group created", conv)
}

func (h *ChatHandler) JoinGroup(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.chatUsecase.JoinGroup(c.Request.Context(), id, userID); err != nil {
		response.Internal(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Joined group", gin.H{})
}

func (h *ChatHandler) MarkRead(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	_ = h.chatUsecase.MarkRead(c.Request.Context(), id, userID)
	response.Success(c, http.StatusOK, "OK", gin.H{"unreadCount": 0})
}

func (h *ChatHandler) DeleteMessage(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BindError(c, err)
		return
	}
	if err := h.chatUsecase.DeleteMessage(c.Request.Context(), id, userID); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Message deleted", gin.H{})
}
