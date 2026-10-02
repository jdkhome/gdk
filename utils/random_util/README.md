# random_util

随机工具包，提供从泛型缓存中随机取数的能力。

## 内容

- `RandomGetFromCache`：从 `github.com/Code-Hex/go-generics-cache` 缓存中按随机键抽取至多 n 条数据。

## 使用示例

```go
package main

import (
    "fmt"
    cache "github.com/Code-Hex/go-generics-cache"
    "github.com/jdkhome/gdk/utils/random_util"
)

func main() {
    c := cache.New[string, int]()
    c.Set("a", 1)
    c.Set("b", 2)
    c.Set("c", 3)

    fmt.Println(random_util.RandomGetFromCache(c, 2))
}
```

## 注意

- 需传入非 nil 缓存；键列表为空或 `n <= 0` 时返回 `nil`。
- `n` 大于键数量时按键数量抽取；读取时已过期或被删除的键会被跳过，不补抽，因此最终数量可能更少。
- 抽样不重复选择键，但不同键的值可能相同。使用 `math/rand`，不适用于安全令牌等密码学用途。
