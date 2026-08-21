# ip_util

IP 地址工具包，用于获取本机出站 IP 及计算 IP 哈希。

## 内容

- `GetOutBoundIP`：通过 UDP 探测获取本机出站 IP。
- `GetIpHash`：获取出站 IP 的 MD5 哈希前 6 位。

## 使用示例

```go
package main

import (
    "fmt"
    "github.com/jdkhome/gdk/utils/ip_util"
)

func main() {
    ip, err := ip_util.GetOutBoundIP()
    fmt.Println(ip, err)
}
```

## 注意

- `GetOutBoundIP` 依赖网络（UDP 连接 `8.8.8.8:53`），离线环境下会返回错误。
- `GetIpHash` 在获取 IP 失败时会 panic。
