// @Author yuzhiwen 1002309  403101988@qq.com
// @Date   2023/11/13 19:12:00
// @Desc   协程控制的实现

package ctxutil

import (
	"context"
	"github.com/subchen/go-trylock/v2"
)

type Broadcaster struct {
	mutex   trylock.TryLocker
	channel chan struct{}
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		mutex:   trylock.New(),
		channel: make(chan struct{}),
	}
}

// Wait   等待管道消息或者Done的消息
// By yuzhiwen 1002309 Date:2023-11-13 19:20:58
func (t *Broadcaster) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return context.DeadlineExceeded
	case <-t.Channel():
		return nil
	}
}

// Channel   锁的方式获取channel
// By yuzhiwen 1002309 Date:2023-11-13 19:21:20
func (t *Broadcaster) Channel() <-chan struct{} {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.channel
}

// Signal  发送信号，唤醒等待信号的协程
// By yuzhiwen 1002309 Date:2023-11-13 19:22:10
func (t *Broadcaster) Signal(ctx context.Context) error {
	if !t.mutex.RTryLock(ctx) {
		return context.DeadlineExceeded
	}
	defer t.mutex.RUnlock()

	select {
	case <-ctx.Done():
		return context.DeadlineExceeded
	case t.channel <- struct{}{}:
	default:
	}

	return nil
}

// Broadcast    广播方式唤醒所有的协程
// By yuzhiwen 1002309 Date:2023-11-13 19:24:57
func (t *Broadcaster) Broadcast(ctx context.Context) error {
	newChannel := make(chan struct{})

	if !t.mutex.TryLock(ctx) {
		return context.DeadlineExceeded
	}
	channel := t.channel
	t.channel = newChannel
	t.mutex.Unlock()

	// send broadcast signal
	close(channel)

	return nil
}
