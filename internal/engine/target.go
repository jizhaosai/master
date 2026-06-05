package engine

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ParsePorts 解析端口范围字符串。
// 支持格式: "1-65535", "80,443", "80,5000-5100" 混合。
// 返回端口集合（map 便于 O(1) 查找）。
func ParsePorts(spec string) (map[uint16]struct{}, error) {
	if spec == "" {
		return nil, fmt.Errorf("端口范围不能为空")
	}

	ports := make(map[uint16]struct{})
	parts := strings.Split(spec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			// 范围格式
			bounds := strings.SplitN(part, "-", 2)
			start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil || start < 1 || start > 65535 {
				return nil, fmt.Errorf("端口范围起始值无效: %s", bounds[0])
			}
			end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil || end < 1 || end > 65535 {
				return nil, fmt.Errorf("端口范围结束值无效: %s", bounds[1])
			}
			if start > end {
				return nil, fmt.Errorf("端口范围起始值(%d)大于结束值(%d)", start, end)
			}
			for p := start; p <= end; p++ {
				ports[uint16(p)] = struct{}{}
			}
		} else {
			// 单端口
			p, err := strconv.Atoi(part)
			if err != nil || p < 1 || p > 65535 {
				return nil, fmt.Errorf("端口无效: %s", part)
			}
			ports[uint16(p)] = struct{}{}
		}
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("未解析到有效端口")
	}
	return ports, nil
}

// ExpandTargets 解析 CIDR / 单 IP / 逗号分隔列表。
// 返回 IP 列表（单播补探用）和网络掩码（归属判断用）。
func ExpandTargets(spec string) ([]net.IP, []*net.IPNet, error) {
	if spec == "" {
		return nil, nil, fmt.Errorf("目标网段不能为空")
	}

	var ips []net.IP
	var nets []*net.IPNet

	parts := strings.Split(spec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "/") {
			// CIDR
			ip, ipnet, err := net.ParseCIDR(part)
			if err != nil {
				return nil, nil, fmt.Errorf("CIDR 格式无效: %s", part)
			}
			nets = append(nets, ipnet)
			// 展开（仅 /24 及以上，避免展开超大网段）
			expanded := expandCIDR(ip, ipnet)
			ips = append(ips, expanded...)
		} else {
			// 单 IP
			ip := net.ParseIP(part)
			if ip == nil {
				return nil, nil, fmt.Errorf("IP 格式无效: %s", part)
			}
			ips = append(ips, ip)
			// 单 IP 也包装成 /32 网络便于后续判断
			mask := net.CIDRMask(32, 32)
			if ip.To4() == nil {
				mask = net.CIDRMask(128, 128)
			}
			nets = append(nets, &net.IPNet{IP: ip.Mask(mask), Mask: mask})
		}
	}
	if len(ips) == 0 {
		return nil, nil, fmt.Errorf("未解析到有效目标")
	}
	return ips, nets, nil
}

// IPInNets 判断 ip 是否落在任一网段内。
func IPInNets(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// expandCIDR 展开 CIDR 为 IP 列表，最多展开 65536 个。
func expandCIDR(ip net.IP, ipnet *net.IPNet) []net.IP {
	var ips []net.IP
	ip = ip.Mask(ipnet.Mask)
	for current := cloneIP(ip); ipnet.Contains(current); incIP(current) {
		ips = append(ips, cloneIP(current))
		if len(ips) >= 65536 {
			break // 防止超大网段
		}
	}
	// 去除网络地址和广播地址（仅 /24 及更小时有意义）
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}
	return ips
}

func cloneIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
