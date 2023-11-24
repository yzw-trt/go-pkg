//go:build !windows
// +build !windows

// @Author yuzhiwen 1002309  403101988@qq.com
// @Date   2023/11/22 14:42:00
// @Desc

package worker

import (
	"context"
	"golang.org/x/sync/errgroup"
	"loghamster/pkg/logger"
	"os"
	"syscall"
)

func startWorker(args []string, attr *syscall.ProcAttr) (pid int, err error) {
	logger.Debug("startWorker进程启动：", logger.String("args[0]", args[0]), logger.Any("args[1:]", args[1:]))
	pid, err = syscall.ForkExec(args[0], args[1:], attr)
	if err != nil {
		logger.Error("启动worker进程失败：", logger.Err(err))
		return
	}
	logger.Info("完成新worker进程启动", logger.Int("pid", pid))
	return
}

func waitWorkers(ctx context.Context, pids []int, args []string, attr *syscall.ProcAttr) error {
	var ws syscall.WaitStatus
	for {
		// wait for any child process
		pid, err := syscall.Wait4(-1, &ws, 0, nil)
		if err != nil {
			logger.Error("wait4() error", logger.Err(err))
			continue
		}
		select {
		case <-ctx.Done():
			return nil
		default:
			// pass
		}
		for i, p := range pids {
			// match our worker's pid
			if p == pid {
				logger.Info("发现进程意外停止", logger.Int("pid", pids[i]), logger.Any("wstatus", ws))
				// only restart once after stopped unexpectedly
				pid, _ = startWorker(args, attr)
				pids[i] = pid
				break
			}
		}
	}
}

func StartWorkers(ctx context.Context, args []string, workerNum int) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	attr := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}

	pids := make([]int, workerNum)
	for i := 0; i < workerNum; i++ {
		pid, err := startWorker(args, attr)
		if err != nil {
			return err
		}
		pids[i] = pid
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return waitWorkers(ctx, pids, args, attr)
	})

	signal := waitSignals(ctx)
	if signal != nil {
		for _, pid := range pids {
			p, err := os.FindProcess(pid)
			if err == nil {
				_ = p.Signal(signal)
			}
		}
	}

	return nil
}

func KillWorker(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	return p.Signal(os.Kill)
}
