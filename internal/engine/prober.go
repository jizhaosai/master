package engine

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"
	"go.uber.org/zap"
	"golang.org/x/net/ipv4"
)

// RawPacket 原始收包数据。
type RawPacket struct {
	Data  []byte
	SrcIP string
}

// Prober mDNS 报文收发器，负责组播/单播查询与监听。
type Prober struct {
	iface     *net.Interface
	listenDur time.Duration
	retries   int
	conn      *net.UDPConn
	pconn     *ipv4.PacketConn
	log       *zap.Logger
	mu        sync.Mutex
}

// mDNS 组播地址
var mdnsGroupAddr = &net.UDPAddr{
	IP:   net.IPv4(224, 0, 0, 251),
	Port: 5353,
}

// NewProber 创建 mDNS 探测器。
// 使用端口复用（SO_REUSEADDR）绑定 5353，确保能接收组播。
func NewProber(iface *net.Interface, listenDur time.Duration, retries int, log *zap.Logger) (*Prober, error) {
	// 使用端口复用方式绑定 5353
	conn, err := reuseListenUDP(iface)
	if err != nil {
		// 降级：如果端口复用也失败，尝试随机端口
		log.Warn("端口复用绑定 5353 失败，尝试随机端口", zap.Error(err))
		conn, err = net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			return nil, fmt.Errorf("创建 UDP socket 失败: %w", err)
		}
		log.Warn("使用随机端口（可能无法接收组播）",
			zap.String("addr", conn.LocalAddr().String()))
	} else {
		log.Info("成功绑定 5353 端口（端口复用模式）",
			zap.String("interface", iface.Name))
	}

	pconn := ipv4.NewPacketConn(conn)

	// 加入组播组
	if err := pconn.JoinGroup(iface, mdnsGroupAddr); err != nil {
		log.Warn("加入组播组失败", zap.Error(err))
	} else {
		log.Info("已加入 mDNS 组播组 224.0.0.251")
	}

	// 设置组播出口网卡
	if err := pconn.SetMulticastInterface(iface); err != nil {
		log.Warn("设置组播出口网卡失败", zap.Error(err))
	}
	_ = pconn.SetMulticastLoopback(true)

	// 允许接收多播包的控制消息
	if err := pconn.SetControlMessage(ipv4.FlagDst, true); err != nil {
		log.Debug("SetControlMessage 失败（非关键）", zap.Error(err))
	}

	return &Prober{
		iface:     iface,
		listenDur: listenDur,
		retries:   retries,
		conn:      conn,
		pconn:     pconn,
		log:       log,
	}, nil
}

// Close 关闭探测器。
func (p *Prober) Close() {
	_ = p.pconn.LeaveGroup(p.iface, mdnsGroupAddr)
	_ = p.conn.Close()
}

// Listen 在监听窗口内持续收包，返回通道。
func (p *Prober) Listen(ctx context.Context) <-chan RawPacket {
	ch := make(chan RawPacket, 512)
	go func() {
		defer close(ch)
		buf := make([]byte, 65535)
		deadline := time.Now().Add(p.listenDur)
		_ = p.conn.SetReadDeadline(deadline)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			n, addr, err := p.conn.ReadFromUDP(buf)
			if err != nil {
				// 超时则正常结束
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					return
				}
				// 其他错误继续
				continue
			}
			data := make([]byte, n)
			copy(data, buf[:n])
			srcIP := ""
			if addr != nil {
				srcIP = addr.IP.String()
			}
			ch <- RawPacket{Data: data, SrcIP: srcIP}
		}
	}()
	return ch
}

// MulticastQuery 发送组播查询（meta-query + 已知服务类型 PTR 查询）。
// 会重发 retries 次，每次间隔递增。
func (p *Prober) MulticastQuery(ctx context.Context, serviceTypes []string) error {
	// 构造 meta-query
	queries := []dns.Question{
		{Name: MetaQueryName, Qtype: dns.TypePTR, Qclass: dns.ClassINET},
	}
	// 附加已知服务类型查询
	for _, st := range serviceTypes {
		queries = append(queries, dns.Question{Name: st, Qtype: dns.TypePTR, Qclass: dns.ClassINET})
	}

	for retry := 0; retry < p.retries; retry++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 每次发送分批（每条消息最多放 10 个 question 避免超包）
		for i := 0; i < len(queries); i += 10 {
			end := i + 10
			if end > len(queries) {
				end = len(queries)
			}
			msg := &dns.Msg{
				MsgHdr: dns.MsgHdr{
					Id:               0, // mDNS 查询 ID 必须为 0 (RFC6762 §18.1)
					RecursionDesired: false,
				},
				Question: queries[i:end],
			}
			data, err := msg.Pack()
			if err != nil {
				p.log.Error("构造查询报文失败", zap.Error(err))
				continue
			}
			p.mu.Lock()
			_, err = p.conn.WriteToUDP(data, mdnsGroupAddr)
			p.mu.Unlock()
			if err != nil {
				p.log.Warn("发送组播查询失败", zap.Error(err))
			}
		}

		// 递增间隔
		delay := time.Duration(200*(retry+1)) * time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return nil
}

// UnicastQuery 对指定 IP 发单播查询（QU bit）。
func (p *Prober) UnicastQuery(ctx context.Context, ip net.IP) error {
	dst := &net.UDPAddr{IP: ip, Port: 5353}
	// 查询 meta-query + 常见类型
	questions := []dns.Question{
		{Name: MetaQueryName, Qtype: dns.TypePTR, Qclass: dns.ClassINET | 1<<15}, // QU bit
	}
	for _, st := range KnownServiceTypes[:5] { // 只查前 5 个减少报文量
		questions = append(questions, dns.Question{Name: st, Qtype: dns.TypePTR, Qclass: dns.ClassINET | 1<<15})
	}

	msg := &dns.Msg{
		MsgHdr: dns.MsgHdr{Id: 0, RecursionDesired: false},
		Question: questions,
	}
	data, err := msg.Pack()
	if err != nil {
		return err
	}
	p.mu.Lock()
	_, err = p.conn.WriteToUDP(data, dst)
	p.mu.Unlock()
	return err
}
