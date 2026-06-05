package engine

import (
	"strings"

	"github.com/miekg/dns"
	"mdns-mapper/internal/model"
)

// ParseMessage 将原始 DNS 报文字节解析为 ResourceRecord 切片。
// srcIP 为报文来源地址，辅助关联。
func ParseMessage(raw []byte, srcIP string) ([]model.ResourceRecord, error) {
	msg := new(dns.Msg)
	if err := msg.Unpack(raw); err != nil {
		return nil, err
	}

	var records []model.ResourceRecord

	// 解析 Answer + Additional + Authority 中的所有记录
	allRR := make([]dns.RR, 0, len(msg.Answer)+len(msg.Extra)+len(msg.Ns))
	allRR = append(allRR, msg.Answer...)
	allRR = append(allRR, msg.Extra...)
	allRR = append(allRR, msg.Ns...)

	for _, rr := range allRR {
		rec := model.ResourceRecord{
			Name:  strings.TrimSuffix(rr.Header().Name, "."),
			Type:  rr.Header().Rrtype,
			TTL:   rr.Header().Ttl,
			SrcIP: srcIP,
		}
		switch v := rr.(type) {
		case *dns.PTR:
			rec.PtrTarget = strings.TrimSuffix(v.Ptr, ".")
		case *dns.SRV:
			rec.SrvTarget = strings.TrimSuffix(v.Target, ".")
			rec.SrvPort = v.Port
			rec.SrvPriority = v.Priority
			rec.SrvWeight = v.Weight
		case *dns.A:
			rec.A = v.A.String()
		case *dns.AAAA:
			rec.AAAA = v.AAAA.String()
		case *dns.TXT:
			kv, raw := parseTXT(v.Txt)
			rec.Txt = kv
			rec.TxtRaw = raw
		default:
			continue // 跳过不关注的类型
		}
		records = append(records, rec)
	}
	return records, nil
}

// parseTXT 解析 TXT 记录中的 key=value 对。
func parseTXT(txt []string) (map[string]string, []string) {
	kv := make(map[string]string, len(txt))
	for _, s := range txt {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if idx := strings.IndexByte(s, '='); idx >= 0 {
			key := s[:idx]
			val := s[idx+1:]
			kv[key] = val
		} else {
			// 布尔型标志
			kv[s] = ""
		}
	}
	return kv, txt
}
