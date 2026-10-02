# guc

Go 并发工具包，提供协程池与安全协程启动能力。

## 内容

- `WorkerPool`：协程池，支持任务提交、等待工作协程退出与任务 panic 恢复。
- `Task`：任务结构，包含上下文、业务标识与执行函数。
- `SafeGo`：启动协程，捕获该协程内执行函数的 panic 并记录日志。

## 使用示例

### 协程池

`Shutdown` 不保证排空队列；需要全部完成时，由调用方先等待任务结束再关闭：

```go
package main

import (
    "sync"

    "github.com/jdkhome/gdk/traces"
    "github.com/jdkhome/gdk/utils/guc"
)

func main() {
    pool := guc.NewWorkerPool(4)
    defer pool.Shutdown()

    var done sync.WaitGroup
    done.Add(1)
    err := pool.Submit(&guc.Task{
        Ctx: traces.NewCtx(),
        Biz: []string{"job", "demo"},
        Fn: func() {
            defer done.Done()
            println("hello")
        },
    })
    if err != nil {
        done.Done()
        panic(err)
    }
    done.Wait()
}
```

### 安全协程

```go
import (
    "context"
    "github.com/jdkhome/gdk/utils/guc"
)

// 在函数中调用；SafeGo 本身不等待任务完成。
guc.SafeGo(context.Background(), []string{"job", "demo"}, func() {
    // 此函数内的可恢复 panic 会被捕获并记录。
})
```

## 生命周期限制

- 协程池大小应为正数；队列容量等于工作协程数，满时 `Submit` 阻塞。
- 关闭前先停止并等待所有提交操作结束，不要并发调用 `Submit` 与 `Shutdown`，否则可能向已关闭的通道发送而 panic。完全关闭后的 `Submit` 返回业务错误。
- `Shutdown` 只能调用一次，等待正在执行的任务退出，但尚未执行的排队任务可能被丢弃；不能从池内任务调用它等待自身退出。
- `Task.Ctx` 用于日志上下文，协程池不会自动根据其取消或超时中止 `Fn`；任务需自行处理取消。
- panic 恢复仅覆盖执行函数所在协程，不覆盖函数自行启动的其他协程；panic 值和堆栈会写入错误日志，避免在 panic 内容中携带敏感信息。
