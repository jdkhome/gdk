# time36

时间与 36 进制字符串互转工具，用于生成紧凑的时间表示。

## 内容

- `TimeToTime36`：将 `time.Time` 转换为 8 位大写 36 进制字符串（以 `2020-01-01 00:00:00 UTC` 为基准的毫秒数）。
- `Time36ToTime`：将 8 位 36 进制字符串解码为基准时间加毫秒偏移，返回 UTC 时间。

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

- `Time36ToTime` 要求输入长度必须为 8 个字节，否则返回错误；字母大小写均可，非法字符返回错误。
- 编码不足 8 位时补零，超过 8 位时只保留末 8 位。只有基准时间起 `0..36^8-1` 毫秒的范围可在毫秒精度下往返，亚毫秒部分会丢失。
- 编码不会拒绝基准时间之前或范围之外的时间；负偏移会转成 uint64，超长编码会截断，不能用于任意时间的无损序列化。
