# model

通用领域模型定义包，当前包含 IP 协议版本枚举。

## 内容

- `IPVersion`：IP 协议版本枚举类型。
- `IPVersion_V4`：IPv4。
- `IPVersion_V6`：IPv6。
- `IPVersions`：枚举集合。

## 使用示例

```go
package main

import "github.com/jdkhome/gdk/model"

func main() {
    if v, ok := model.IPVersions.GetByCode("ipv4"); ok {
        println(v.Value.Name)
    }
}
```
