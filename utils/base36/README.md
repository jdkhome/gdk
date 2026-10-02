# base36

36 进制编解码工具，编码字符集为 `0-9` 与 `A-Z`，解码兼容小写字母。

## 内容

- `Uint64ToBase36`：将 `uint64` 转换为大写 36 进制字符串，零编码为 `"0"`。
- `Base36ToUint64`：将 36 进制字符串转换为 `uint64`，遇到非法字符或溢出返回错误；空字符串返回 `0, nil`，不去除空白，也不接受正负号。

## 使用示例

```go
package main

import (
    "fmt"
    "github.com/jdkhome/gdk/utils/base36"
)

func main() {
    s := base36.Uint64ToBase36(1296)
    fmt.Println(s) // 100

    n, err := base36.Base36ToUint64("100")
    fmt.Println(n, err) // 1296 <nil>
}
```
