package netutil

import (
	"fmt"
	"net"
	"strings"
)

// PickInterface 根据名称选择网卡。
// 若 name 为空，自动选第一个 Up 且有组播能力的网卡。
func PickInterface(name string) (*net.Interface, error) {
	if name != "" {
		iface, err := net.InterfaceByName(name)
		if err != nil {
			return nil, fmt.Errorf("找不到网卡 %s: %w", name, err)
		}
		return iface, nil
	}

	// 自动选择：优先有组播能力、状态为 Up、有单播地址的
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("枚举网卡失败: %w", err)
	}

	// 第一遍：优先选物理网卡（排除虚拟网卡：vEthernet、Hyper-V、VPN、Docker 等）
	var fallback *net.Interface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		if len(addrs) == 0 {
			continue
		}
		// 判断是否为虚拟网卡
		if isVirtualInterface(iface.Name) {
			if fallback == nil {
				selected := iface
				fallback = &selected
			}
			continue
		}
		// 物理网卡优先返回
		selected := iface
		return &selected, nil
	}
	// 没有物理网卡则用虚拟的
	if fallback != nil {
		return fallback, nil
	}
	return nil, fmt.Errorf("未找到可用的组播网卡")
}

// ListInterfaces 列出所有活跃且有组播能力的网卡（供前端展示）。
func ListInterfaces() ([]InterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []InterfaceInfo
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		if len(addrs) == 0 {
			continue
		}
		info := InterfaceInfo{
			Name:  iface.Name,
			Index: iface.Index,
		}
		for _, addr := range addrs {
			info.Addrs = append(info.Addrs, addr.String())
		}
		result = append(result, info)
	}
	return result, nil
}

// InterfaceInfo 网卡基本信息。
type InterfaceInfo struct {
	Name  string   `json:"name"`
	Index int      `json:"index"`
	Addrs []string `json:"addrs"`
}

// isVirtualInterface 判断网卡名是否为虚拟网卡。
func isVirtualInterface(name string) bool {
	// 常见虚拟网卡关键词
	virtualKeywords := []string{
		"vEthernet", "Hyper-V", "VPN", "Docker", "VMware",
		"VirtualBox", "WSL", "Loopback", "isatap", "Teredo",
	}
	lower := strings.ToLower(name)
	for _, kw := range virtualKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}
