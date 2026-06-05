//go:build !windows

package engine

import (
	"context"
	"fmt"
	"net"
	"syscall"
)

// reuseListenUDP 在 Linux/macOS 上使用 SO_REUSEADDR + SO_REUSEPORT 绑定 UDP 5353。
func reuseListenUDP(iface *net.Interface) (*net.UDPConn, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			err := c.Control(func(fd uintptr) {
				// SO_REUSEADDR
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				if opErr != nil {
					return
				}
				// SO_REUSEPORT (值为 15，Linux 和 macOS 均支持)
				opErr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, 0x0F, 1)
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
