package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd 根命令。
var rootCmd = &cobra.Command{
	Use:   "mdns-mapper",
	Short: "mDNS 资产测绘工具",
	Long:  "一款基于 mDNS/DNS-SD 协议的网络资产测绘 CLI 工具，支持 IP 网段扫描、深度 banner 识别与 MySQL 持久化。",
}

// Execute 执行根命令。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "configs/config.yaml", "配置文件路径")
}
