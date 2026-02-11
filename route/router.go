package route

import (
	"ai-agent/api"
	"ai-agent/service"
	"github.com/gin-gonic/gin"
)

func Register(r *gin.Engine, chatSvc *service.ChatService) {

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 聊天接口分组
	chatGroup := r.Group("/chat")
	{
		chatGroup.POST("", api.ChatHandler(chatSvc)) // POST /chat

	}

	// 以后你可以继续加分组
	// adminGroup := r.Group("/admin")
	// bizGroup := r.Group("/biz")
}
