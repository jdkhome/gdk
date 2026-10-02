# tx

循环计数器与按 ID 保存数据的内存事务管理工具；不提供数据库提交、回滚或持久化。

## 内容

- `GlobalCounter` / `NewGlobalCounter(min, max)`：线程安全的循环自增计数器，支持 `Next`、`Reset`、`Current`、`Set`。
- `NextDefault`：使用默认计数器获取下一个值，范围为 1 到 1000（含端点）。
- `TransactionManager[T]` / `NewTransactionManager[T](maxID)`：支持事务数据的创建、获取、释放与按回调清理。

## 使用示例

### 计数器

```go
package main

import (
    "fmt"
    "github.com/jdkhome/gdk/tx"
)

func main() {
    counter := tx.NewGlobalCounter(1, 1000)
    fmt.Println(counter.Next()) // 1
    fmt.Println(counter.Next()) // 2
}
```

`min` 必须小于 `max`，否则 panic。`max` 小于 uint64 最大值时，达到 `max` 后下一次 `Next` 回绕到 `min`；`max` 为 uint64 最大值时则溢出为 0，非零下界场景应避免该上界。初始化或 `Reset` 后 `Current` 为 `min-1`（`min=0` 时为 uint64 最大值），下一次 `Next` 才返回 `min`。`Set` 只接受闭区间内的值，越界返回错误。

### 事务管理器

```go
package main

import "github.com/jdkhome/gdk/tx"

func main() {
    manager := tx.NewTransactionManager[int](100)
    id := manager.CreateTransaction(42)
    if v, ok := manager.GetTransaction(id); ok {
        println(v)
    }
    manager.ReleaseTransaction(id)
}
```

- ID 范围为 `0..maxID`（含端点），所以示例最多容纳 101 条同时存在的数据，而不是 100 条；`maxID` 不得为 uint64 最大值（内部 `maxID+1` 会溢出）。
- ID 全占用时，`CreateTransaction` 持续等待重试，直到有 ID 被释放；没有超时或取消参数。
- `CleanupExpiredTransactions(isExpired func(T) bool)` 需调用方主动执行，并自行定义过期规则；没有内置 TTL 或后台清理。
