// @Author yuzhiwen 1002309  403101988@qq.com
// @Date   2023/11/21 16:58:00
// @Desc

package ctxutil

import (
	"context"
	"os"
	"os/signal"
)

// ContextWithOSSignal   ctx捕获系统信号，用作退出处理
// By yuzhiwen 1002309 Date:2023-11-21 16:59:49
func ContextWithOSSignal(parent context.Context, sig ...os.Signal) context.Context {
	osSignalChan := make(chan os.Signal, 1)
	signal.Notify(osSignalChan, sig...)

	ctx, cancel := context.WithCancel(parent)

	go func(cancel context.CancelFunc) {
		<-osSignalChan
		cancel()
	}(cancel)

	return ctx
}
