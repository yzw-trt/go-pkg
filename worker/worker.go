// @Author yuzhiwen 1002309  403101988@qq.com
// @Date   2023/11/22 14:25:00
// @Desc

package worker

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// waitSignals   注册并等待获取系统信号
// By yuzhiwen 1002309 Date:2023-11-22 14:16:15
func waitSignals(ctx context.Context) os.Signal {
	osSignalChan := make(chan os.Signal, 1)
	signal.Notify(osSignalChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		return nil
	case sig := <-osSignalChan:
		return sig
	}
}
