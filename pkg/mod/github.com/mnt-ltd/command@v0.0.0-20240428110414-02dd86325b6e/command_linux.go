package command

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// ExecCommand 执行cmd命令操作
func execCommand(name string, args []string, opt ...Option) (out string, err error) {
	var (
		stderr, stdout bytes.Buffer
		expire         = 30 * time.Minute
		errs           []string
	)

	if len(opt) > 0 && opt[0].Timeout != nil {
		expire = *opt[0].Timeout
	}

	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if len(opt) > 0 && opt[0].Stdout != nil {
		cmd.Stdout = opt[0].Stdout
	} else {
		cmd.Stdout = &stdout
	}

	if len(opt) > 0 && opt[0].Stderr != nil {
		cmd.Stderr = opt[0].Stderr
	} else {
		cmd.Stderr = &stderr
	}

	err = cmd.Start()
	if err != nil {
		err = fmt.Errorf("%s\n%s", err.Error(), stderr.String())
		return
	}

	pid := 0
	if cmd.Process != nil && cmd.Process.Pid != 0 {
		pid = cmd.Process.Pid
		pidMap.Store(pid, pid)
		if len(opt) > 0 && opt[0].Callback != nil {
			opt[0].Callback(pid)
		}
	}
	defer func() {
		if pid != 0 {
			pidMap.Delete(pid)
		}
	}()

	time.AfterFunc(expire, func() {
		if cmd.Process != nil && cmd.Process.Pid != 0 {
			errs = append(errs, fmt.Sprintf("execute timeout: %d min.", int(expire.Minutes())))
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			cmd.Process.Kill()
		}
	})

	err = cmd.Wait()
	if err != nil {
		errs = append(errs, err.Error(), stderr.String())
	}
	out = stdout.String()
	if len(errs) > 0 {
		errs = append(errs, out)
		err = errors.New(strings.Join(errs, "\n\r"))
	}
	return
}

func closeProcess(pid int) {
	fmt.Println("kill pid:", pid)
	syscall.Kill(-pid, syscall.SIGKILL)
	pidMap.Delete(pid)
}
