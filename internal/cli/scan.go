package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"mdns-mapper/internal/config"
	"mdns-mapper/internal/logger"
	"mdns-mapper/internal/model"
	"mdns-mapper/internal/output"
	"mdns-mapper/internal/repository"
	"mdns-mapper/internal/service"
)

var (
	scanCIDR        string
	scanPorts       string
	scanInterface   string
	scanListen      int
	scanRetries     int
	scanUnicast     bool
	scanConcurrency int
	scanFormat      string
	scanOutput      string
	scanSaveDB      bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "执行 mDNS 资产扫描",
	Long:  "对指定 IP 网段进行 mDNS/DNS-SD 资产发现，输出资产信息与深度 banner。",
	RunE:  runScan,
}

func init() {
	scanCmd.Flags().StringVar(&scanCIDR, "cidr", "", "目标 IP 网段 (必填，如 192.168.1.0/24)")
	scanCmd.Flags().StringVar(&scanPorts, "ports", "1-65535", "端口范围 (如 80,443,5000-5100)")
	scanCmd.Flags().StringVar(&scanInterface, "interface", "", "指定出口网卡名")
	scanCmd.Flags().IntVar(&scanListen, "listen", 5, "组播监听窗口(秒)")
	scanCmd.Flags().IntVar(&scanRetries, "retries", 3, "查询重发次数")
	scanCmd.Flags().BoolVar(&scanUnicast, "unicast", true, "是否单播补探")
	scanCmd.Flags().IntVar(&scanConcurrency, "concurrency", 256, "单播并发数")
	scanCmd.Flags().StringVar(&scanFormat, "format", "text", "输出格式 (text/json)")
	scanCmd.Flags().StringVar(&scanOutput, "output", "", "输出文件路径(默认标准输出)")
	scanCmd.Flags().BoolVar(&scanSaveDB, "save-db", false, "是否写入 MySQL")

	_ = scanCmd.MarkFlagRequired("cidr")
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	defer log.Sync()

	// 构建扫描输入
	input := service.ScanInput{
		CIDR:        scanCIDR,
		Ports:       scanPorts,
		Interface:   scanInterface,
		ListenDur:   time.Duration(scanListen) * time.Second,
		Retries:     scanRetries,
		Unicast:     scanUnicast,
		Concurrency: scanConcurrency,
	}

	// 执行扫描
	scanSvc := service.NewScanService(log)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(scanListen+60)*time.Second)
	defer cancel()

	result, err := scanSvc.Run(ctx, input)
	if err != nil {
		return fmt.Errorf("扫描失败: %w", err)
	}

	// 格式化输出
	var text string
	switch scanFormat {
	case "json":
		text, err = output.FormatJSON(result.Assets)
		if err != nil {
			return fmt.Errorf("JSON 格式化失败: %w", err)
		}
	default:
		text = output.FormatText(result.Assets)
	}

	// 写输出
	if scanOutput != "" {
		if err := os.WriteFile(scanOutput, []byte(text), 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}
		fmt.Printf("结果已写入: %s\n", scanOutput)
	} else {
		fmt.Print(text)
	}

	// 持久化到数据库
	if scanSaveDB {
		db, err := repository.NewDB(cfg.MySQL)
		if err != nil {
			return fmt.Errorf("连接数据库失败: %w", err)
		}
		taskRepo := repository.NewScanTaskRepo(db)
		task := &model.ScanTask{
			CIDR:      scanCIDR,
			PortRange: scanPorts,
			Interface: scanInterface,
			Status:    "done",
			CreatedAt: time.Now(),
		}
		if err := taskRepo.Create(task); err != nil {
			return fmt.Errorf("创建任务记录失败: %w", err)
		}
		if err := taskRepo.SaveAssets(task.ID, result.Assets); err != nil {
			return fmt.Errorf("保存资产失败: %w", err)
		}
		_ = taskRepo.UpdateAssetCount(task.ID, result.Total)
		fmt.Printf("已保存到数据库，任务 ID: %d，资产数: %d\n", task.ID, result.Total)
	}

	fmt.Printf("\n扫描完成，发现 %d 个资产\n", result.Total)
	return nil
}
