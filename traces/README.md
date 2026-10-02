# traces

链路追踪包，用于生成 traceID / spanID 并写入 `context.Context`。

## 内容

- `Tracer`：追踪器，使用 `GetTraceID` / `GetSpanID` 读取标识。
- `NewTracer`：生成新的追踪器；traceID 由 8 位时间、6 位 IP 哈希、6 位进程级随机串与 4 位循环自增序号组成，初始 spanID 为 `"0"`。
- `WithTracer(ctx)`：基于传入上下文创建并写入一个新追踪器，不接受已有 `Tracer`，也不继承原 traceID。
- `NewCtx()`：在 `context.Background()` 上调用 `WithTracer`。
- `GetTracer(ctx)`：从 context 读取追踪器，没有时返回 nil；需传入非 nil context。

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

## 注意

- 当前没有创建子 span 或设置外部 traceID 的公开 API。
- 本包实际使用字符串键 `"CTX#TRACE"`，不是 `ctxs.Trace` 枚举键；通过后者存入的值不会被 `GetTracer` 读取。
- 包初始化会调用 `ip_util.GetIpHash`；获取出站地址失败会 panic，间接依赖本包的 `logs` 等包也受影响。traceID 不是密码学随机标识，不应当作安全令牌。
