# guc

Go 并发工具包，提供协程池与安全协程启动能力。

## 内容

- `WorkerPool`：协程池，支持任务提交、优雅关闭与 panic 恢复。
- `Task`：任务结构，包含上下文、业务标识与执行函数。
- `SafeGo`：安全启动协程，捕获 panic 并记录日志，避免影响主进程。

## 使用示例

### 协程池

```go
package main

import (
    "github.com/jdkhome/gdk/traces"
    "github.com/jdkhome/gdk/utils/guc"
)

func main() {
    pool := guc.NewWorkerPool(4)
    defer pool.Shutdown()

    _ = pool.Submit(&guc.Task{
        Ctx: traces.NewCtx(),
        Biz: []string{"job", "demo"},
        Fn:  func() { println("hello") },
    })
}
```

### 安全协程

```go
import (
    "context"
    "github.com/jdkhome/gdk/utils/guc"
)

guc.SafeGo(context.Background(), []string{"job", "demo"}, func() {
    // 即使发生 panic 也不会导致进程崩溃
})
```
