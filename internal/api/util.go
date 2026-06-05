package api

import (
	"context"
	"time"
)

// contextWithTimeout 创建带超时的 context 和 cancel 函数。
// 调用方负责在任务结束时调用 cancel。
func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
