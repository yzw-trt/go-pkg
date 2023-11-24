//go:build windows
// +build windows

// @Author yuzhiwen 1002309  403101988@qq.com
// @Date   2023/11/22 14:19:00
// @Desc   worker进程管理工具,windows实现

package worker

import (
	"context"
	"golang.org/x/sync/errgroup"
	"loghamster/pkg/logger"
	"os"
	"syscall"
	"unsafe"
)

func startWorker(args []string, attr *syscall.ProcAttr) (pid int, handle uintptr, err error) {
	logger.Debug("startWorker进程启动：", logger.String("args[0]", args[0]), logger.Any("args[1:]", args[1:]))
	pid, handle, err = syscall.StartProcess(args[0], args[1:], attr)
	if err != nil {
		logger.Error("启动worker进程失败：", logger.Err(err))
		return
	}
	logger.Info("完成新worker进程启动", logger.Int("pid", pid))
	return
}

// waitWorkers
// By yuzhiwen 1002309 Date:2023-11-22 14:28:51
func waitWorkers(ctx context.Context, pids []int, handles []uintptr, args []string, attr *syscall.ProcAttr) error {
	// syscall only has `WaitForSingleObject`, but we have to wait multiple processes,
	// so that we find proc `WaitForMultipleObjects` from kernel32.dll.
	// doc: https://docs.microsoft.com/en-us/windows/desktop/api/synchapi/nf-synchapi-waitformultipleobjects
	dll := syscall.MustLoadDLL("kernel32.dll")
	defer dll.Release()
	wfmo := dll.MustFindProc("WaitForMultipleObjects")

	for {
		r1, _, err := wfmo.Call(uintptr(len(handles)), uintptr(unsafe.Pointer(&handles[0])), 0, syscall.INFINITE)
		ret := int(r1)
		if ret == syscall.WAIT_FAILED && err != nil {
			logger.Error("WaitForMultipleObjects()", logger.Err(err))
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			// pass
		}
		if ret >= syscall.WAIT_OBJECT_0 && ret < syscall.WAIT_OBJECT_0+len(handles) {
			i := ret - syscall.WAIT_OBJECT_0
			syscall.CloseHandle(syscall.Handle(handles[i]))
			logger.Info("发现进程意外停止", logger.Int("pid", pids[i]))
			// only restart once after stopped unexpectedly
			pid, handle, _ := startWorker(args, attr)
			pids[i] = pid
			handles[i] = handle
		}
	}
}

// StartWorkers   创建多个worker进程，args参数需要给到 attr暂时写死，也可以放到外部获取
// By yuzhiwen 1002309 Date:2023-11-22 14:32:19
func StartWorkers(ctx context.Context, args []string, workerNum int, pids []int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 进程运行的环境变量和文件句柄
	attr := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}

	// 创建线程并记录PID和句柄
	// pids := make([]int, workerNum)
	handles := make([]uintptr, workerNum)
	for i := 0; i < workerNum; i++ {
		pid, handle, err := startWorker(args, attr)
		if err != nil {
			return err
		}
		pids[i] = pid
		handles[i] = handle
	}

	// 创建协程来监控由这里创建的进程
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return waitWorkers(ctx, pids, handles, args, attr) // 监控进程
	})

	// 阻塞等待系统退出信号或者ctx取消信号,就kill掉创建的所有进程
	signal := waitSignals(ctx)
	if signal != nil {
		for _, pid := range pids {
			p, err := os.FindProcess(pid)
			if err == nil {
				// Sending Interrupt on Windows is not implemented. Look at https://github.com/golang/go/issues/6720 for more info
				_ = p.Signal(os.Kill)
			}
		}
	}
	return nil
}

// KillWorker   kill 指定pid的进程
// By yuzhiwen 1002309 Date:2023-11-22 15:00:44
func KillWorker(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	
	return p.Signal(os.Kill)
}
