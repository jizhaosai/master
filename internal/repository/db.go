package repository

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"mdns-mapper/internal/config"
	"mdns-mapper/internal/model"
)

// NewDB 根据配置创建 GORM 数据库连接，并自动迁移表结构。
// 如果数据库不存在则自动创建。
func NewDB(cfg config.MySQLConfig) (*gorm.DB, error) {
	// 先连接不带数据库名的地址，尝试创建数据库
	if err := ensureDatabase(cfg); err != nil {
		return nil, err
	}

	// 再连接到指定数据库
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(&model.ScanTask{}, &model.Asset{}, &model.Service{}); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}
	return db, nil
}

// ensureDatabase 确保目标数据库存在，不存在则自动创建。
func ensureDatabase(cfg config.MySQLConfig) error {
	// 不带数据库名的 DSN
	charset := cfg.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=%s&parseTime=%t&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, charset, cfg.ParseTime)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("连接 MySQL 服务器失败: %w", err)
	}

	// 创建数据库（如果不存在）
	createSQL := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci",
		cfg.Database,
	)
	if err := db.Exec(createSQL).Error; err != nil {
		return fmt.Errorf("创建数据库失败: %w", err)
	}

	// 关闭连接
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		_ = sqlDB.Close()
	}
	return nil
}
