# gdk

Go 开发工具包（Go Development Kit），提供一组可复用的基础能力，覆盖错误处理、日志、枚举、链路追踪、并发工具等常见场景。

## 安装

```bash
go get github.com/jdkhome/gdk
```

要求 Go 1.24+。

## 子包

| 包 | 说明 |
| --- | --- |
| [di](di/README.md) | 编译期依赖注入代码生成，支持泛型接口绑定、泛型 provider 与 list 注入 |
| [enums](enums/README.md) | 泛型枚举构建器，提供类型安全的 code/value 枚举 |
| [errs](errs/README.md) | 业务错误定义、包装与判断 |
| [logs](logs/README.md) | 分级日志，支持控制台与文件输出 |
| [traces](traces/README.md) | 链路追踪，生成 traceID / spanID |
| [tx](tx/README.md) | 循环自增计数器与事务管理器 |
| [ctxs](ctxs/README.md) | 上下文键定义 |
| [model](model/README.md) | 通用领域模型（含 IP 版本枚举） |
| [model/i18n](model/i18n/README.md) | 国际化字符串类型 |
| [utils/base36](utils/base36/README.md) | 36 进制编解码 |
| [utils/time36](utils/time36/README.md) | 时间与 36 进制互转 |
| [utils/guc](utils/guc/README.md) | 协程池与安全协程 |
| [utils/ip_util](utils/ip_util/README.md) | 出站 IP 获取与哈希 |
| [utils/json_util](utils/json_util/README.md) | 对象转 JSON 字符串 |
| [utils/map_utils](utils/map_utils/README.md) | 泛型读取 map 值 |
| [utils/random_util](utils/random_util/README.md) | 从缓存中随机取数 |

## 快速开始

```go
package main

import (
    "github.com/jdkhome/gdk/errs"
    "github.com/jdkhome/gdk/logs"
    "github.com/jdkhome/gdk/traces"
)

func main() {
    ctx := traces.NewCtx()
    logs.Info(ctx, []string{"demo", "start"}, "服务启动")

    err := errs.NewBizErr([]string{"demo", "create"}, "创建失败: %s", "参数错误")
    if errs.Is(err, errs.BizErr) {
        logs.Error(ctx, []string{"demo", "create"}, err.Error())
    }
}
```

## 测试

```bash
go test ./...
```
