package main

import (
	"log"

	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/internal/server"
	"github.com/yockii/dify_tools/pkg/config"
	"github.com/yockii/dify_tools/pkg/database"
	"github.com/yockii/dify_tools/pkg/logger"
	"github.com/yockii/dify_tools/pkg/util"
)

func main() {
	// 初始化配置
	if err := config.Init(); err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}

	util.InitNode(config.GetUint64("server.node_id"))

	// 初始化日志
	logger.Init()

	// 连接数据库
	database.Init()

	// 数据库迁移
	model.AutoMigrate(database.GetDB())

	model.InitData(database.GetDB())

	// 创建服务器实例
	srv := server.New()

	// 启动服务器
	if err := srv.Start(); err != nil {
		log.Fatalf("服务停止: %v", err)
	}
}
