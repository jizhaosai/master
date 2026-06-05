package output

import (
	"fmt"
	"strings"

	"mdns-mapper/internal/model"
)

// FormatText 将资产列表格式化为需求示例同款的文本格式。
func FormatText(assets []model.Asset) string {
	var b strings.Builder

	for _, asset := range assets {
		b.WriteString("services:\n")
		// 已收集的服务类型用于 answers 段
		var serviceTypes []string

		for _, svc := range asset.Services {
			b.WriteString(svc.Banner)
			serviceTypes = append(serviceTypes, svc.ServiceType)
		}

		// answers 段
		b.WriteString("answers:\n")
		b.WriteString("PTR:\n")
		for _, st := range serviceTypes {
			fmt.Fprintf(&b, "    %s.local\n", st)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// FormatJSON 将资产列表序列化为 JSON（直接用 encoding/json）。
// 这里放到 json.go 里实现。
