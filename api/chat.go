package api

import (
	"ai-agent/model"
	"ai-agent/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ChatHandler(chatSvc *service.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.ChatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		resp, err := chatSvc.HandleMessage(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func IntentRecognitionHandler(chatSvc *service.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req model.IntentRecognitionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		resp, err := chatSvc.RecognizeIntent(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func CreateTicketHandler(chatSvc *service.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserID      string `json:"user_id"`
			SessionID   string `json:"session_id"`
			Description string `json:"description"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		ticket, err := chatSvc.CreateTicket(c.Request.Context(), req.UserID, req.SessionID, req.Description)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, ticket)
	}
}

func SessionHistoryHandler(chatSvc *service.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("session_id")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}

		history, err := chatSvc.GetSessionHistory(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, history)
	}
}

func ClearSessionHandler(chatSvc *service.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("session_id")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}

		if err := chatSvc.ClearSession(c.Request.Context(), sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "session cleared"})
	}
}
