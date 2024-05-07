// @Author yuzhiwen 1002309  403101988@qq.com
// @Date   2023/11/14 14:02:00
// @Desc  sleep功能

package ctxutil

import (
	"context"
	"time"
)

func Sleep(ctx context.Context, duration time.Duration) (done bool) {
	if duration < 1 {
		return false
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return true
	case <-timer.C:
		return false
	}
}
