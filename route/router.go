package route

import (
	"ai-agent/api"
	"ai-agent/service"
	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine, chatSvc *service.ChatService) {

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	chatGroup := r.Group("/chat")
	{
		chatGroup.POST("", api.ChatHandler(chatSvc))
	}

	intentGroup := r.Group("/intent")
	{
		intentGroup.POST("/recognize", api.IntentRecognitionHandler(chatSvc))
	}

	ticketGroup := r.Group("/ticket")
	{
		ticketGroup.POST("/create", api.CreateTicketHandler(chatSvc))
	}
}
