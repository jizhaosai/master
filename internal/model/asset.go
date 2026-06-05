package model

import "time"

// ScanTask 扫描任务，记录一次 mDNS 资产测绘的输入与执行状态。
type ScanTask struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	CIDR       string    `gorm:"column:cidr;size:64;not null" json:"cidr"`               // 输入的 IP 网段
	PortRange  string    `gorm:"column:port_range;size:128;not null" json:"port_range"` // 输入的端口范围
	Interface  string    `gorm:"column:interface;size:64" json:"interface"`             // 使用的网卡
	Status     string    `gorm:"size:16;not null;default:pending" json:"status"`        // pending/running/done/failed
	AssetCount int       `gorm:"not null;default:0" json:"asset_count"`                 // 发现资产数
	ErrMsg     string    `gorm:"column:err_msg;size:512" json:"err_msg,omitempty"`      // 失败原因
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`

	Assets []Asset `gorm:"foreignKey:TaskID" json:"assets,omitempty"`
}

// TableName 指定扫描任务表名。
func (ScanTask) TableName() string { return "scan_task" }

// Asset 资产（主机维度），一台主机在一次扫描中的发现结果。
type Asset struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID      uint64    `gorm:"index;not null" json:"task_id"`
	Host        string    `gorm:"size:255;index" json:"host"`              // 主机名 *.local
	IPv4        string    `gorm:"column:ipv4;size:45;index" json:"ipv4"`   // IPv4 地址
	IPv6        string    `gorm:"column:ipv6;size:64" json:"ipv6"`         // IPv6 地址
	MAC         string    `gorm:"column:mac;size:32" json:"mac"`           // MAC（若实例名含）
	DeviceModel string    `gorm:"size:128" json:"device_model"`            // device-info.model 设备型号
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`

	Services []Service `gorm:"foreignKey:AssetID" json:"services,omitempty"`
}

// TableName 指定资产表名。
func (Asset) TableName() string { return "asset" }

// Service 服务实例（DNS-SD service instance），承载 banner 深度内容。
type Service struct {
	ID           uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	AssetID      uint64            `gorm:"index;not null" json:"asset_id"`
	ServiceType  string            `gorm:"size:128;index" json:"service_type"`  // 如 _http._tcp
	InstanceName string            `gorm:"size:255" json:"instance_name"`       // 服务实例名 Name=
	Port         uint16            `gorm:"index" json:"port"`                   // SRV 端口
	Protocol     string            `gorm:"size:8;default:tcp" json:"protocol"`  // tcp/udp
	Priority     uint16            `json:"priority"`
	Weight       uint16            `json:"weight"`
	TTL          uint32            `json:"ttl"`
	TxtRecords   map[string]string `gorm:"-" json:"txt_records"`                // 内存态 TXT 键值对
	TxtRecordsJSON string          `gorm:"column:txt_records;type:json" json:"-"` // 入库的 JSON 串
	Banner       string            `gorm:"type:text" json:"banner"`             // 格式化后的可读 banner
}

// TableName 指定服务表名。
func (Service) TableName() string { return "service" }
