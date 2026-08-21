# ctxs

上下文（context）键定义包，基于 `enums` 提供的枚举能力，统一管理与 `context.Context` 相关的键。

## 内容

- `Key`：上下文键类型，底层为 `enums.Member[enums.Value]`。
- `Trace`：链路追踪键，code 为 `CTX#TRACE`。
- `Keys`：所有已注册上下文键的枚举集合。

## 使用示例

```go
import "github.com/jdkhome/gdk/ctxs"

// 判断某个键是否为已注册的合法键
if ctxs.Keys.CheckMember(ctxs.Trace) {
    // ...
}

// 通过 code 获取对应键
key, ok := ctxs.Keys.GetByCode("CTX#TRACE")
```
