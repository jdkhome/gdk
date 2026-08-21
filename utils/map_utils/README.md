# map_utils

Map 工具包，提供泛型读取 map 值的能力。

## 内容

- `GetValue[T]`：从 `map[string]any` 中按 key 读取并断言为指定类型 `T`。

## 使用示例

```go
package main

import (
    "fmt"
    "github.com/jdkhome/gdk/utils/map_utils"
)

func main() {
    m := map[string]any{"age": 18}

    if age, ok := map_utils.GetValue[int](m, "age"); ok {
        fmt.Println(age) // 18
    }
}
```
