package command

import (
	"os"
	"sync"
	"time"
)

type Option struct {
	Timeout  *time.Duration
	Stdout   *os.File
	Stderr   *os.File
	Callback func(pid int)
}

// pidMap 用于存储所有的子进程pid，以便在主程序退出时，kill掉所有的相关子进程
var pidMap sync.Map

// 当主程序退出时，从pidMap中获取所有的pid，然后kill掉
func CloseChildProccess(pid ...int) {
	if len(pid) > 0 {
		for _, p := range pid {
			closeProcess(p)
		}
		return
	}

	pidMap.Range(func(key, value interface{}) bool {
		if pid, ok := value.(int); ok {
			closeProcess(pid)
		}
		return true
	})
}

// ExecCommand 执行cmd命令操作
func ExecCommand(name string, args []string, timeout ...time.Duration) (out string, err error) {
	opt := Option{}
	if len(timeout) > 0 {
		opt.Timeout = &timeout[0]
	}
	return execCommand(name, args, opt)
}

func ExecCommandV2(name string, args []string, opt Option) (out string, err error) {
	return execCommand(name, args, opt)
}
