# logs

日志包，提供分级日志、多输出（控制台 / 文件）以及默认全局日志器。

## 内容

- `Level`：日志级别枚举（`Level_Debug`、`Level_Info`、`Level_Warn`、`Level_Error`）。
- `Output`：日志输出接口，定义 `PushLog`。
- `CategoryOutput`：可选接口，额外接收业务分类（`biz`），用于按分类路由。
- `ConsoleOutput`：控制台输出。
- `FileOutput`：基于 lumberjack 的文件输出（支持大小、保留时间、备份数）。
- `RoutedOutput`：按业务分类首段过滤后转发，`ExcludeBiz` / `IncludeBiz` 用于剥离或定向分类。
- `NewBizFileOutput`：仅写入指定分类的滚动文件输出。
- `Logger`：日志器，根据最低级别过滤并分发到多个输出。
- 全局便捷函数：`Info`、`Debug`、`Warn`、`Error`。

## 使用示例

```go
package main

import (
    "github.com/jdkhome/gdk/logs"
    "github.com/jdkhome/gdk/traces"
)

func main() {
    ctx := traces.NewCtx()
    logs.Info(ctx, []string{"order", "create"}, "订单创建成功, id=%d", 1001)
    logs.Error(ctx, []string{"order", "pay"}, "支付失败: %s", "余额不足")
}
```

按分类路由到独立文件，并从通用输出剥离高噪声分类：

```go
logs.DefaultLogger = logs.NewLogger(logs.Level_Info, []logs.Output{
    logs.ExcludeBiz(logs.Console, "db_sql"),
    logs.ExcludeBiz(logs.AppDefaultLog, "db_sql"),
    logs.CommonError,
    logs.NewBizFileOutput(logs.Level_Info, []string{"db_sql"}, "./logs/db_sql.log", 10, 7, 5),
})
```

## 注意

- 日志默认输出到 `./logs/app_default.log` 与 `./logs/common_error.log`。
- 低于日志器配置级别的日志会被直接丢弃。
- `RoutedOutput` 只能路由通过 `Logger.push` 传入分类的日志；直接调用 `PushLog`（无分类）时会被丢弃，避免高噪声日志误入汇总。
