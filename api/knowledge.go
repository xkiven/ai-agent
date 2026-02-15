package api

import (
	"ai-agent/model"
	"ai-agent/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AddKnowledgeRequest struct {
	Texts    []string         `json:"texts"`
	Metadata []map[string]any `json:"metadata,omitempty"`
}

type AddKnowledgeResponse struct {
	Success bool   `json:"success"`
	Count   int    `json:"count"`
	Message string `json:"message"`
}

func AddKnowledgeHandler(chatSvc *service.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req AddKnowledgeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		pythonReq := model.KnowledgeRequest{
			Texts:    req.Texts,
			Metadata: req.Metadata,
		}

		resp, err := chatSvc.CallPythonKnowledgeAdd(pythonReq)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

type ListKnowledgeResponse struct {
	Success bool            `json:"success"`
	Data    []KnowledgeItem `json:"data"`
	Total   int             `json:"total"`
	Message string          `json:"message"`
}

type KnowledgeItem struct {
	Text     string `json:"text"`
	Category string `json:"category,omitempty"`
}

func ListKnowledgeHandler(chatSvc *service.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := chatSvc.CallPythonKnowledgeList()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

type DeleteKnowledgeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func DeleteKnowledgeHandler(chatSvc *service.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		index := c.Query("index")
		if index == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "index is required"})
			return
		}

		resp, err := chatSvc.CallPythonKnowledgeDelete(index)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func ClearKnowledgeHandler(chatSvc *service.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := chatSvc.CallPythonKnowledgeClear()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

type CountKnowledgeResponse struct {
	Success bool   `json:"success"`
	Count   int    `json:"count"`
	Message string `json:"message"`
}

func KnowledgeCountHandler(chatSvc *service.ChatService) gin.HandlerFunc {
	return func(c *gin.Context) {
		resp, err := chatSvc.CallPythonKnowledgeCount()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}
