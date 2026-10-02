# ip_util

IP 地址工具包，用于获取本机出站 IP 及计算 IP 哈希。

## 内容

- `GetOutBoundIP`：通过 UDP socket 的本地地址获取系统为指定目标选择的出站 IP，不是公网 IP 查询。
- `GetIpHash`：获取出站 IP 的 MD5 十六进制哈希前 6 位；不是安全身份凭证，也不保证唯一。

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

- `GetOutBoundIP` 使用 `net.Dial("udp", "8.8.8.8:53")` 后读取本地地址，不发送探测数据，不验证远端连通性。依赖系统路由 / socket 可用性；没有互联网也可能成功，创建连接失败时返回错误。
- `GetIpHash` 在获取 IP 失败时会 panic。
