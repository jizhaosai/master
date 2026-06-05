//go:build windows

package engine

import (
	"context"
	"fmt"
	"net"
	"syscall"

	"golang.org/x/sys/windows"
)

// reuseListenUDP 在 Windows 上使用 SO_REUSEADDR 绑定 UDP 5353 端口。
// 通过 net.ListenConfig 的 Control 回调在 socket 创建后、绑定前设置选项。
func reuseListenUDP(iface *net.Interface) (*net.UDPConn, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				// 设置 SO_REUSEADDR 允许多进程共享端口
				opErr = windows.SetsockoptInt(
					windows.Handle(fd),
					windows.SOL_SOCKET,
					windows.SO_REUSEADDR,
					1,
				)
			})
			if err != nil {
				return err
			}
			return opErr
		},
	}

	pc, err := lc.ListenPacket(context.Background(), "udp4", "0.0.0.0:5353")
	if err != nil {
		return nil, fmt.Errorf("端口复用绑定 5353 失败: %w", err)
	}

	udpConn, ok := pc.(*net.UDPConn)
	if !ok {
		pc.Close()
		return nil, fmt.Errorf("类型断言 UDPConn 失败")
	}

	return udpConn, nil
}
