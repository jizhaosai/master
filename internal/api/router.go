package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"mdns-mapper/internal/repository"
	"mdns-mapper/internal/service"
)

// SetupRouter 配置 Gin 路由。
func SetupRouter(db *gorm.DB, scanSvc *service.ScanService, log *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	// 允许跨域（前端开发用）
	r.Use(corsMiddleware())

	// 静态文件服务（前端页面）
	r.Static("/static", "./web/static")
	r.StaticFile("/", "./web/index.html")
	r.StaticFile("/favicon.ico", "./web/favicon.ico")

	// 注入依赖
	taskRepo := repository.NewScanTaskRepo(db)
	handler := NewHandler(scanSvc, taskRepo, log)

	// API 路由组
	api := r.Group("/api/v1")
	{
		// 扫描相关
		api.POST("/scans", handler.CreateScan)
		api.GET("/scans", handler.ListScans)
		api.GET("/scans/:id", handler.GetScan)
		api.GET("/scans/:id/assets", handler.GetScanAssets)
		api.GET("/scans/:id/export", handler.ExportScan)

		// 网卡列表（前端选择用）
		api.GET("/interfaces", handler.ListInterfaces)
	}

	return r
}

// corsMiddleware 简单 CORS 中间件。
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
