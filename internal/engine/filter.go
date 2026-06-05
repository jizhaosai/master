package engine

import (
	"net"

	"mdns-mapper/internal/model"
)

// FilterAssets 按 IP 网段和端口范围过滤资产及其服务。
// 仅保留 IPv4 落在目标网段 且 至少有一个服务端口落在端口集合内的资产。
func FilterAssets(assets []model.Asset, nets []*net.IPNet, ports map[uint16]struct{}) []model.Asset {
	var result []model.Asset

	for _, asset := range assets {
		// IP 过滤：检查 IPv4 是否在目标网段内
		if asset.IPv4 != "" {
			ip := net.ParseIP(asset.IPv4)
			if ip != nil && !IPInNets(ip, nets) {
				continue // 不在目标网段
			}
		}

		// 端口过滤：保留端口落在集合内的服务
		// 如果端口集合包含所有端口（1-65535），则不过滤
		if len(ports) < 65535 {
			var filteredServices []model.Service
			for _, svc := range asset.Services {
				if svc.Port == 0 {
					// 无端口的服务（如 device-info）始终保留
					filteredServices = append(filteredServices, svc)
					continue
				}
				if _, ok := ports[svc.Port]; ok {
					filteredServices = append(filteredServices, svc)
				}
			}
			if len(filteredServices) == 0 {
				continue // 无匹配端口的服务
			}
			asset.Services = filteredServices
		}

		result = append(result, asset)
	}
	return result
}
