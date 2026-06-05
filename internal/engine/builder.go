package engine

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/miekg/dns"
	"mdns-mapper/internal/model"
)

// macRegex 从 workstation 实例名中提取 MAC 地址。
var macRegex = regexp.MustCompile(`\[([0-9a-fA-F:]{17})\]`)

// BuildAssets 将零散的 ResourceRecord 按 host 关联重组为 Asset 列表。
func BuildAssets(records []model.ResourceRecord) []model.Asset {
	// 索引结构
	type srvInfo struct {
		target   string
		port     uint16
		priority uint16
		weight   uint16
		ttl      uint32
	}

	// 按实例名索引 SRV
	srvByInstance := make(map[string]srvInfo)
	// 按实例名索引 TXT
	txtByInstance := make(map[string]map[string]string)
	txtRawByInstance := make(map[string][]string)
	// 按 host 索引 A/AAAA
	aByHost := make(map[string]string)
	aaaByHost := make(map[string]string)
	// PTR: 服务类型 → 实例名列表
	ptrByType := make(map[string][]string)
	// 设备信息
	deviceInfoTxt := make(map[string]map[string]string) // host → txt

	for _, r := range records {
		switch r.Type {
		case dns.TypePTR:
			// 过滤 meta-query 的应答（服务类型→服务类型）与普通 PTR（服务类型→实例名）
			if r.Name == "_services._dns-sd._udp.local" {
				// meta-query 应答：ptrTarget 是服务类型
				continue
			}
			ptrByType[r.Name] = append(ptrByType[r.Name], r.PtrTarget)

		case dns.TypeSRV:
			srvByInstance[r.Name] = srvInfo{
				target:   r.SrvTarget,
				port:     r.SrvPort,
				priority: r.SrvPriority,
				weight:   r.SrvWeight,
				ttl:      r.TTL,
			}

		case dns.TypeTXT:
			txtByInstance[r.Name] = r.Txt
			txtRawByInstance[r.Name] = r.TxtRaw

		case dns.TypeA:
			aByHost[r.Name] = r.A

		case dns.TypeAAAA:
			aaaByHost[r.Name] = r.AAAA
		}
	}

	// 重组：遍历每个服务类型下的实例
	// host → Asset
	assetMap := make(map[string]*model.Asset)

	for serviceType, instances := range ptrByType {
		for _, instanceName := range instances {
			srv, hasSrv := srvByInstance[instanceName]
			txt := txtByInstance[instanceName]

			host := ""
			var port uint16
			var priority, weight uint16
			var ttl uint32
			if hasSrv {
				host = srv.target
				port = srv.port
				priority = srv.priority
				weight = srv.weight
				ttl = srv.ttl
			}

			// 若没有 SRV，尝试从源 IP 推断
			if host == "" {
				// 回退：用实例名中可能的 host 信息
				continue
			}

			// 获取或创建 Asset
			asset, ok := assetMap[host]
			if !ok {
				ipv4 := aByHost[host]
				ipv6 := aaaByHost[host]
				now := time.Now()
				asset = &model.Asset{
					Host:      host,
					IPv4:      ipv4,
					IPv6:      ipv6,
					FirstSeen: now,
					LastSeen:  now,
				}
				assetMap[host] = asset
			}

			// 提取 MAC（从 workstation 实例名）
			if strings.Contains(serviceType, "_workstation") {
				if matches := macRegex.FindStringSubmatch(instanceName); len(matches) > 1 {
					asset.MAC = matches[1]
				}
			}

			// 提取 device-info
			if strings.Contains(serviceType, "_device-info") {
				deviceInfoTxt[host] = txt
				if m, ok := txt["model"]; ok {
					asset.DeviceModel = m
				}
			}

			// 构建 Service
			// 提取短服务类型（去除 _tcp.local / _udp.local 后缀）
			protocol := "tcp"
			if strings.Contains(serviceType, "_udp") {
				protocol = "udp"
			}

			// 实例名的 Name 部分（去掉服务类型后缀）
			displayName := extractInstanceDisplayName(instanceName, serviceType)

			svc := model.Service{
				ServiceType:  serviceType,
				InstanceName: displayName,
				Port:         port,
				Protocol:     protocol,
				Priority:     priority,
				Weight:       weight,
				TTL:          ttl,
				TxtRecords:   txt,
			}

			// 生成 banner
			svc.Banner = buildBanner(&svc, asset)
			// 序列化 TxtRecords 为 JSON 用于入库
			if txt != nil {
				jsonBytes, _ := json.Marshal(txt)
				svc.TxtRecordsJSON = string(jsonBytes)
			}

			asset.Services = append(asset.Services, svc)
		}
	}

	// 补充 device-info 中的 model
	for host, txt := range deviceInfoTxt {
		if asset, ok := assetMap[host]; ok {
			if m, exists := txt["model"]; exists && asset.DeviceModel == "" {
				asset.DeviceModel = m
			}
		}
	}

	// 转为切片
	assets := make([]model.Asset, 0, len(assetMap))
	for _, a := range assetMap {
		assets = append(assets, *a)
	}
	return assets
}

// extractInstanceDisplayName 从实例全名中提取用户友好的显示名。
// 实例全名格式：InstanceName._serviceType._proto.local
func extractInstanceDisplayName(instanceName, serviceType string) string {
	// 去掉服务类型后缀
	suffix := "." + serviceType
	name := strings.TrimSuffix(instanceName, suffix)
	// 有时 local 后缀也在
	name = strings.TrimSuffix(name, ".local")
	return name
}

// buildBanner 生成示例同款格式的 banner 文本。
func buildBanner(svc *model.Service, asset *model.Asset) string {
	var b strings.Builder
	shortType := shortServiceType(svc.ServiceType)
	fmt.Fprintf(&b, "%d/%s %s:\n", svc.Port, svc.Protocol, shortType)
	fmt.Fprintf(&b, "    Name=%s\n", svc.InstanceName)
	fmt.Fprintf(&b, "    IPv4=%s\n", asset.IPv4)
	fmt.Fprintf(&b, "    IPv6=%s\n", asset.IPv6)
	fmt.Fprintf(&b, "    Hostname=%s\n", asset.Host)
	fmt.Fprintf(&b, "    TTL=%d\n", svc.TTL)
	// TXT 深度键值对
	for k, v := range svc.TxtRecords {
		if v != "" {
			fmt.Fprintf(&b, "    %s=%s\n", k, v)
		}
	}
	return b.String()
}

// shortServiceType 把 "_http._tcp.local" 简化为 "http"。
func shortServiceType(st string) string {
	s := strings.TrimPrefix(st, "_")
	if idx := strings.Index(s, "."); idx > 0 {
		s = s[:idx]
	}
	return s
}
