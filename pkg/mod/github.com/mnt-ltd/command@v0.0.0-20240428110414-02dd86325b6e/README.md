# command

对`os/exec`的简易封装，防止出现`孤儿进程`

## 使用示例

### 执行相应指令

```go
command.ExecCommand("ebook-convert", []string{"1.txt","1.pdf"}, 30*time.Minute)
```

### 关闭可能存在的孤儿进程
```go
...
c := make(chan os.Signal, 1)
signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
go func() {
    s := <-c
    fmt.Println("get signal：", s)
    fmt.Println("close child process...")
    command.CloseChildProccess()
    fmt.Println("close child process done.")
    fmt.Println("exit.")
    os.Exit(0)
}()
...
```