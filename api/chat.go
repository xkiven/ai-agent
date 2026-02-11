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
