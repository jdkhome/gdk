# traces

链路追踪包，用于生成 traceID / spanID 并写入 `context.Context`。

## 内容

- `Tracer`：追踪器，保存 `traceID` 与 `spanID`。
- `NewTracer`：生成新的追踪器（traceID 由时间、IP 哈希、随机串、自增序号组成）。
- `WithTracer` / `NewCtx`：将追踪器写入 context。
- `GetTracer`：从 context 中读取追踪器。

## 使用示例

```go
package main

import (
    "fmt"
    "github.com/jdkhome/gdk/traces"
)

func main() {
    ctx := traces.NewCtx()
    if tracer := traces.GetTracer(ctx); tracer != nil {
        fmt.Println(tracer.GetTraceID(), tracer.GetSpanID())
    }
}
```
