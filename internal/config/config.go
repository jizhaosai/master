package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config 应用全局配置。
type Config struct {
	Scan   ScanConfig   `mapstructure:"scan"`
	MySQL  MySQLConfig  `mapstructure:"mysql"`
	Server ServerConfig `mapstructure:"server"`
	Log    LogConfig    `mapstructure:"log"`
}

// ScanConfig 扫描相关配置。
type ScanConfig struct {
	ListenSecs  int  `mapstructure:"listen_secs"`  // 组播监听窗口（秒）
	Retries     int  `mapstructure:"retries"`      // 查询重发次数
	Unicast     bool `mapstructure:"unicast"`      // 是否单播补探
	Concurrency int  `mapstructure:"concurrency"`  // 单播并发数
}

// ListenDuration 返回监听窗口的 time.Duration。
func (s ScanConfig) ListenDuration() time.Duration {
	if s.ListenSecs <= 0 {
		return 5 * time.Second
	}
	return time.Duration(s.ListenSecs) * time.Second
}

// MySQLConfig 数据库配置。
type MySQLConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	User      string `mapstructure:"user"`
	Password  string `mapstructure:"password"`
	Database  string `mapstructure:"database"`
	Charset   string `mapstructure:"charset"`
	ParseTime bool   `mapstructure:"parse_time"`
}

// DSN 拼接 MySQL 连接串。
func (m MySQLConfig) DSN() string {
	charset := m.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=Local",
		m.User, m.Password, m.Host, m.Port, m.Database, charset, m.ParseTime)
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Addr string `mapstructure:"addr"`
}

// LogConfig 日志配置。
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load 从指定路径加载配置文件，并支持环境变量覆盖（如 ${MYSQL_PASSWORD}）。
func Load(path string) (*Config, error) {
	v := viper.New()
	setDefaults(v)

	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			// 配置文件不存在时使用默认值，不报错
			if !os.IsNotExist(err) {
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return nil, fmt.Errorf("读取配置文件失败: %w", err)
				}
			}
		}
	}

	// 允许环境变量覆盖
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 数据库密码优先读环境变量
	if pwd := os.Getenv("MYSQL_PASSWORD"); pwd != "" {
		cfg.MySQL.Password = pwd
	}
	return &cfg, nil
}

// setDefaults 设置默认配置值。
func setDefaults(v *viper.Viper) {
	v.SetDefault("scan.listen_secs", 5)
	v.SetDefault("scan.retries", 3)
	v.SetDefault("scan.unicast", true)
	v.SetDefault("scan.concurrency", 256)

	v.SetDefault("mysql.host", "127.0.0.1")
	v.SetDefault("mysql.port", 3306)
	v.SetDefault("mysql.user", "root")
	v.SetDefault("mysql.database", "mdns_mapper")
	v.SetDefault("mysql.charset", "utf8mb4")
	v.SetDefault("mysql.parse_time", true)

	v.SetDefault("server.addr", ":8080")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
}
