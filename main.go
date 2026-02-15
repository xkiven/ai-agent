package main

import (
	"ai-agent/dao"
	"ai-agent/internal/aiclient"
	"ai-agent/model"
	"ai-agent/route"
	"ai-agent/service"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"time"
)

func main() {
	r := gin.Default()

	aiClient := aiclient.NewClient("http://127.0.0.1:8000")

	intentConfig, err := loadIntentConfig("config/intents.yaml")
	if err != nil {
		log.Fatalf("加载意图配置失败: %v", err)
	}
	log.Printf("加载意图配置成功，共 %d 个意图", len(intentConfig.Intents))

	store := dao.NewRedisStore("localhost:6379", "", 0, 24*time.Hour)
	chatSvc := service.NewChatService(aiClient, store, intentConfig.Intents)

	route.Register(r, chatSvc)

	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}

func loadIntentConfig(path string) (*model.IntentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config model.IntentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &config, nil
}
