-- mDNS 资产测绘系统数据库初始化脚本
-- 使用方法: mysql -u root -p < scripts/init_db.sql

CREATE DATABASE IF NOT EXISTS mdns_mapper
  DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

USE mdns_mapper;

-- 扫描任务表
CREATE TABLE IF NOT EXISTS scan_task (
    id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    cidr          VARCHAR(64)  NOT NULL          COMMENT '输入的IP网段',
    port_range    VARCHAR(128) NOT NULL          COMMENT '输入的端口范围',
    `interface`   VARCHAR(64)  DEFAULT NULL      COMMENT '使用的网卡',
    status        VARCHAR(16)  NOT NULL DEFAULT 'pending' COMMENT 'pending/running/done/failed',
    asset_count   INT          NOT NULL DEFAULT 0 COMMENT '发现资产数',
    err_msg       VARCHAR(512) DEFAULT NULL      COMMENT '失败原因',
    started_at    DATETIME     DEFAULT NULL,
    finished_at   DATETIME     DEFAULT NULL,
    created_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_status (status),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='mDNS扫描任务';

-- 资产表（主机维度）
CREATE TABLE IF NOT EXISTS asset (
    id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    task_id       BIGINT UNSIGNED NOT NULL       COMMENT '所属任务',
    host          VARCHAR(255) DEFAULT NULL      COMMENT '主机名 *.local',
    ipv4          VARCHAR(45)  DEFAULT NULL,
    ipv6          VARCHAR(64)  DEFAULT NULL,
    mac           VARCHAR(32)  DEFAULT NULL,
    device_model  VARCHAR(128) DEFAULT NULL      COMMENT 'device-info.model 设备型号',
    first_seen    DATETIME     NOT NULL,
    last_seen     DATETIME     NOT NULL,
    INDEX idx_task (task_id),
    INDEX idx_host (host),
    INDEX idx_ipv4 (ipv4),
    CONSTRAINT fk_asset_task FOREIGN KEY (task_id) REFERENCES scan_task(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='mDNS资产(主机)';

-- 服务表（服务实例维度，承载 banner 深度内容）
CREATE TABLE IF NOT EXISTS service (
    id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    asset_id      BIGINT UNSIGNED NOT NULL,
    service_type  VARCHAR(128) DEFAULT NULL      COMMENT '如 _http._tcp',
    instance_name VARCHAR(255) DEFAULT NULL      COMMENT '服务实例名',
    port          SMALLINT UNSIGNED DEFAULT NULL COMMENT 'SRV端口',
    protocol      VARCHAR(8)   DEFAULT 'tcp',
    priority      SMALLINT UNSIGNED DEFAULT 0,
    weight        SMALLINT UNSIGNED DEFAULT 0,
    ttl           INT UNSIGNED DEFAULT 0,
    txt_records   JSON         DEFAULT NULL      COMMENT 'TXT键值对(banner深度)',
    banner        TEXT         DEFAULT NULL      COMMENT '格式化banner',
    INDEX idx_asset (asset_id),
    INDEX idx_port (port),
    INDEX idx_type (service_type),
    CONSTRAINT fk_service_asset FOREIGN KEY (asset_id) REFERENCES asset(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='mDNS服务实例';
