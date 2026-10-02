# ctxs

上下文（context）键定义包，基于 `enums` 提供的枚举能力定义键类型。

## 内容

- `Key`：上下文键类型，底层为 `enums.Member[enums.Value]`。
- `Trace`：追踪用途的枚举键，code 为 `CTX#TRACE`。
- `Keys`：本包已注册上下文键的枚举集合。

## 使用示例

```go
package main

import "github.com/jdkhome/gdk/ctxs"

func main() {
    println(ctxs.Keys.CheckMember(ctxs.Trace)) // true
    if key, ok := ctxs.Keys.GetByCode("CTX#TRACE"); ok {
        println(key.Code)
    }
}
```

## 注意

`ctxs.Trace` 是枚举结构体，不等于字符串 `"CTX#TRACE"`。当前 `traces` 包内部使用后者，因此不要通过 `context.WithValue(ctx, ctxs.Trace, ...)` 向 `traces.GetTracer` 传值；请使用 `traces.WithTracer` 或 `traces.NewCtx`。
