package service

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"

	"mdns-mapper/internal/engine"
	"mdns-mapper/internal/model"
	"mdns-mapper/pkg/netutil"
)

// ScanInput 扫描输入参数。
type ScanInput struct {
	CIDR        string        // 目标网段
	Ports       string        // 端口范围
	Interface   string        // 网卡名称
	ListenDur   time.Duration // 监听窗口
	Retries     int           // 重发次数
	Unicast     bool          // 是否单播补探
	Concurrency int           // 单播并发数
}

// ScanResult 扫描结果。
type ScanResult struct {
	Assets []model.Asset `json:"assets"`
	Total  int           `json:"total"`
}

// ScanService 扫描编排服务。
type ScanService struct {
	log *zap.Logger
}

// NewScanService 创建扫描服务。
func NewScanService(log *zap.Logger) *ScanService {
	return &ScanService{log: log}
}

// Run 执行 mDNS 资产测绘扫描。
func (s *ScanService) Run(ctx context.Context, in ScanInput) (*ScanResult, error) {
	// 1. 解析输入
	ips, nets, err := engine.ExpandTargets(in.CIDR)
	if err != nil {
		return nil, fmt.Errorf("解析目标网段失败: %w", err)
	}
	ports, err := engine.ParsePorts(in.Ports)
	if err != nil {
		return nil, fmt.Errorf("解析端口范围失败: %w", err)
	}

	// 2. 选网卡
	iface, err := netutil.PickInterface(in.Interface)
	if err != nil {
		return nil, fmt.Errorf("选择网卡失败: %w", err)
	}
	s.log.Info("开始扫描",
		zap.String("cidr", in.CIDR),
		zap.String("ports", in.Ports),
		zap.String("interface", iface.Name),
		zap.Duration("listen", in.ListenDur),
	)

	// 3. 创建探测器
	prober, err := engine.NewProber(iface, in.ListenDur, in.Retries, s.log)
	if err != nil {
		return nil, fmt.Errorf("创建探测器失败: %w", err)
	}
	defer prober.Close()

	// 4. 启动监听
	pktCh := prober.Listen(ctx)

	// 5. 组播查询（异步）
	go func() {
		if err := prober.MulticastQuery(ctx, engine.KnownServiceTypes); err != nil {
			s.log.Warn("组播查询异常", zap.Error(err))
		}
	}()

	// 6. 单播补探（可选，异步）
	if in.Unicast && len(ips) > 0 {
		go s.fanOutUnicast(ctx, prober, ips, in.Concurrency)
	}

	// 7. 收包解析
	var records []model.ResourceRecord
	for pkt := range pktCh {
		rrs, err := engine.ParseMessage(pkt.Data, pkt.SrcIP)
		if err != nil {
			s.log.Debug("报文解析失败", zap.Error(err))
			continue
		}
		records = append(records, rrs...)
	}
	s.log.Info("收包完成", zap.Int("records", len(records)))

	// 8. 重组
	assets := engine.BuildAssets(records)
	s.log.Info("资产重组完成", zap.Int("assets_before_filter", len(assets)))

	// 9. 过滤
	assets = engine.FilterAssets(assets, nets, ports)
	s.log.Info("过滤后", zap.Int("assets", len(assets)))

	return &ScanResult{
		Assets: assets,
		Total:  len(assets),
	}, nil
}

// fanOutUnicast 对目标 IP 列表做单播补探。
func (s *ScanService) fanOutUnicast(ctx context.Context, prober *engine.Prober, ips []net.IP, concurrency int) {
	if concurrency <= 0 {
		concurrency = 256
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(targetIP net.IP) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := prober.UnicastQuery(ctx, targetIP); err != nil {
				s.log.Debug("单播查询失败", zap.String("ip", targetIP.String()), zap.Error(err))
			}
		}(ip)
	}
	wg.Wait()
}
