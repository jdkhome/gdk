# errs

业务错误定义与包装工具包，提供带 code 的错误类型以及错误判断、包装能力。

## 内容

- `Err`：带 `Code` 与 `Msg` 的错误结构，`*Err` 实现 `error` 接口，格式为 `[code]msg`。
- `DefErr`：定义一个 `*Err`。
- `Is(err, targets ...*Err)`：用 `errors.As` 找到首个 `*Err`，按 `Code` 与任一目标比较（不比较指针或消息）。不是遍历匹配所有 `*Err`；目标需非 nil。
- `Wrapf`：以 `[biz]msg -> err` 形式通过 `%w` 包装底层错误，业务分类用 `|` 连接。
- `NewBizErr` / `NewTimeout`：便捷构造业务错误与超时错误。
- 预定义错误：`UnknownErr`、`BizErr`、`Timeout`。

## 使用示例

```go
package main

import "github.com/jdkhome/gdk/errs"

func main() {
    err := errs.NewBizErr([]string{"order", "create"}, "创建订单失败: %d", 1001)

    if errs.Is(err, errs.BizErr) {
        println(err.Error())
    }
}
```

输出示例：

```text
[order|create]创建订单失败: 1001 -> [biz_error]业务错误
```
