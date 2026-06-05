package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"mdns-mapper/internal/api"
	"mdns-mapper/internal/config"
	"mdns-mapper/internal/logger"
	"mdns-mapper/internal/repository"
	"mdns-mapper/internal/service"
)

var serveAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动 Web 服务",
	Long:  "启动 Gin HTTP 服务，提供 REST API 和前端页面。",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "", "监听地址 (默认 :8080)")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	defer log.Sync()

	// 连接数据库
	db, err := repository.NewDB(cfg.MySQL)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 创建扫描服务
	scanSvc := service.NewScanService(log)

	// 监听地址优先级：命令行 > 配置文件
	addr := cfg.Server.Addr
	if serveAddr != "" {
		addr = serveAddr
	}

	// 启动路由
	router := api.SetupRouter(db, scanSvc, log)
	fmt.Printf("mDNS 资产测绘服务启动，监听 %s\n", addr)
	fmt.Printf("前端页面: http://localhost%s\n", addr)
	fmt.Printf("API 文档: http://localhost%s/api/v1/\n", addr)
	return router.Run(addr)
}
