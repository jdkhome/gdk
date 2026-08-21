# random_util

随机工具包，提供从泛型缓存中随机取数的能力。

## 内容

- `RandomGetFromCache`：从 `github.com/Code-Hex/go-generics-cache` 缓存中随机获取 n 条数据。

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

- 缓存为空或 `n <= 0` 时返回 `nil`。
- `n` 大于缓存数量时，按缓存实际数量返回。
