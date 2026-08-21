# tx

事务与计数器相关工具包。

## 内容

- `GlobalCounter`：线程安全的循环自增计数器，支持 `Next`、`Reset`、`Current`、`Set`。
- `NextDefault`：使用默认计数器获取下一个值。
- `TransactionManager[T]`：基于 ID 的事务管理器，支持事务的创建、获取、释放与过期清理。

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
