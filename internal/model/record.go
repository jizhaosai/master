package model

// ResourceRecord 是 mDNS 报文解析后的中间态资源记录。
// 它把 DNS 报文里的 PTR/SRV/TXT/A/AAAA 等记录统一抽象，
// 便于后续 AssetBuilder 按 host 与服务实例名做关联重组。
type ResourceRecord struct {
	// Name 资源记录名称（已去除结尾点的 FQDN）
	Name string
	// Type DNS 记录类型，对应 miekg/dns 的 dns.TypePTR / TypeSRV 等
	Type uint16
	// TTL 资源记录存活时间（秒）
	TTL uint32
	// SrcIP 应答报文来源 IP，单播补探时用于辅助关联
	SrcIP string

	// PtrTarget PTR 记录指向的目标（服务类型或服务实例名）
	PtrTarget string

	// SrvTarget SRV 记录的目标主机名（即 Hostname）
	SrvTarget string
	// SrvPort SRV 记录声明的服务监听端口
	SrvPort uint16
	// SrvPriority SRV 优先级
	SrvPriority uint16
	// SrvWeight SRV 权重
	SrvWeight uint16

	// A IPv4 地址（A 记录）
	A string
	// AAAA IPv6 地址（AAAA 记录）
	AAAA string

	// Txt 解析后的 TXT 键值对，banner 深度识别的核心数据
	Txt map[string]string
	// TxtRaw TXT 记录的原始字符串切片
	TxtRaw []string
}
