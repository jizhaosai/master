package output

import (
	"encoding/json"

	"mdns-mapper/internal/model"
)

// FormatJSON 将资产列表序列化为缩进 JSON 字符串。
func FormatJSON(assets []model.Asset) (string, error) {
	data, err := json.MarshalIndent(assets, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
