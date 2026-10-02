# map_utils

Map 工具包，提供泛型读取 map 值的能力。

## 内容

- `GetValue[T]`：从 `map[string]any` 中按 key 读取并通过 Go 类型断言转换为 `T`。nil map、键不存在或断言失败时，返回 `T` 的零值和 false；不做数字、字符串等类型转换。

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
