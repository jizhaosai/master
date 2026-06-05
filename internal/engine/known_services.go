package engine

// KnownServiceTypes 常见的 DNS-SD 服务类型。
// mDNS 组播查询时除了发 meta-query 外，也直接查这些已知类型以提高覆盖率。
var KnownServiceTypes = []string{
	"_http._tcp.local.",
	"_https._tcp.local.",
	"_smb._tcp.local.",
	"_afpovertcp._tcp.local.",
	"_ssh._tcp.local.",
	"_ftp._tcp.local.",
	"_printer._tcp.local.",
	"_ipp._tcp.local.",
	"_workstation._tcp.local.",
	"_device-info._tcp.local.",
	"_airplay._tcp.local.",
	"_raop._tcp.local.",
	"_googlecast._tcp.local.",
	"_qdiscover._tcp.local.",
	"_nfs._tcp.local.",
	"_rdp._tcp.local.",
	"_telnet._tcp.local.",
	"_mqtt._tcp.local.",
}

// MetaQueryName DNS-SD 元查询的名称，用于枚举所有服务类型。
const MetaQueryName = "_services._dns-sd._udp.local."
