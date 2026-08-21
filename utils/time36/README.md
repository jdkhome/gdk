# time36

时间与 36 进制字符串互转工具，用于生成紧凑的时间表示。

## 内容

- `TimeToTime36`：将 `time.Time` 转换为 8 位 36 进制字符串（以 `2020-01-01 00:00:00 UTC` 为基准的毫秒数）。
- `Time36ToTime`：将 8 位 36 进制字符串转换回 `time.Time`。

## 使用示例

```go
package main

import (
    "fmt"
    "time"
    "github.com/jdkhome/gdk/utils/time36"
)

func main() {
    s := time36.TimeToTime36(time.Now())
    fmt.Println(s)

    t, err := time36.Time36ToTime(s)
    fmt.Println(t, err)
}
```

## 注意

- `Time36ToTime` 要求输入长度必须为 8 位，否则返回错误。
