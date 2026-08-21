# json_util

JSON 工具包，提供对象转 JSON 字符串的能力。

## 内容

- `ToJsonStr`：将任意值序列化为 JSON 字符串，禁用 HTML 转义；序列化失败时返回空字符串。

## 使用示例

```go
package main

import (
    "fmt"
    "github.com/jdkhome/gdk/utils/json_util"
)

func main() {
    fmt.Println(json_util.ToJsonStr(map[string]int{"a": 1})) // {"a":1}
}
```
