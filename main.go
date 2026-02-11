package main

import (
	"ai-agent/internal/aiclient"
	"ai-agent/route"
	"ai-agent/service"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	aiClient := aiclient.NewClient("http://127.0.0.1:8000")
	// 初始化 service
	chatSvc := service.NewChatService(aiClient)

	// 注册路由
	route.Register(r, chatSvc)

	// 启动服务
	if err := r.Run(":8080"); err != nil {
		panic(err)
	}

}
