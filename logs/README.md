# logs

日志包，提供分级日志、多输出（控制台 / 文件）、分类路由以及默认全局日志器。

## 内容

- `Level` / `Levels`：日志级别与枚举集合（`Level_Debug`、`Level_Info`、`Level_Warn`、`Level_Error`）。
- `Output`：日志输出接口，定义 `PushLog`。
- `CategoryOutput`：可选接口，额外接收业务分类（`biz`），用于按分类路由。
- `ConsoleOutput` / `NewConsoleOutput`：控制台输出（使用 Go 内建 `println`，写入 stderr）。
- `FileOutput` / `NewFileOutput`：基于 lumberjack 的滚动文件输出；`maxSize` 单位为 MB、`maxAge` 为天、`maxBackups` 为备份数，备份文件名使用 UTC，不压缩。
- `RoutedOutput`：按业务分类首段过滤后转发，`ExcludeBiz` / `IncludeBiz` 用于剥离或定向分类。
- `NewBizFileOutput`：仅写入指定分类的滚动文件输出。
- `Logger` / `NewLogger`：根据最低级别过滤并同步、依次分发到多个输出。
- 全局便捷函数：`Info`、`Debug`、`Warn`、`Error`；`SetDefaultLogger` 用于替换默认日志器。
- `WithAgent` / `AgentRef`：写入和读取身份的 SHA-256 十六进制哈希，不在上下文中保存原始身份；空身份保持原上下文。
- `WithRun` / `RunID` / `NewRunID`：写入、读取和生成运行 ID；生成值为 16 字节密码学随机数的 32 位十六进制表示，随机源失败时 panic。

## 使用示例

```go
package main

import (
    "github.com/jdkhome/gdk/logs"
    "github.com/jdkhome/gdk/traces"
)

func main() {
    ctx := traces.NewCtx()
    ctx = logs.WithAgent(ctx, "order-worker")
    ctx = logs.WithRun(ctx, logs.NewRunID())
    logs.Info(ctx, []string{"order", "create"}, "订单创建成功, id=%d", 1001)
    logs.Error(ctx, []string{"order", "pay"}, "支付失败: %s", "余额不足")
}
```

每条日志包含时间、级别、traceID / spanID、业务分类、正文及进程级随机 `instance_id`；上下文存在身份或运行 ID 时追加 `agent_ref` / `run_id`。`run_id` 使用带引号的转义格式，避免控制字符注入日志字段。`WithAgent` / `WithRun` 需要非 nil 上下文；日志方法接受 nil 上下文，缺少追踪器时 traceID / spanID 为空。

按分类路由到独立文件，并从通用输出剥离高噪声分类：

```go
logs.SetDefaultLogger(logs.NewLogger(logs.Level_Info, []logs.Output{
    logs.ExcludeBiz(logs.Console, "db_sql"),
    logs.ExcludeBiz(logs.AppDefaultLog, "db_sql"),
    logs.CommonError,
    logs.NewBizFileOutput(logs.Level_Info, []string{"db_sql"}, "./logs/db_sql.log", 10, 7, 5),
}))
```

此例仍将 `db_sql` 的 Error 日志写入 `CommonError`。路由按 `biz[0]` 精确匹配，不做字符串前缀匹配；`IncludeBiz` 的空筛选列表不限制分类。无分类的 `PushLog` 调用会被 `RoutedOutput` 丢弃；应通过日志器或 `CategoryOutput.PushLogCategory` 传递分类。路由包装器向内层调用普通 `PushLog`，不要嵌套路由包装器。

## 默认级别与并发

- 默认日志器最低级别为 Info；`Console` 与 `AppDefaultLog` 为 Info，`CommonError` 为 Error。Debug 默认不输出；开启 Debug 需同时调整日志器和目标输出的级别。
- 默认文件路径相对于进程工作目录：`./logs/app_default.log`、`./logs/common_error.log`，均使用 `10 MB / 7 天 / 5 个备份` 配置。
- `SetDefaultLogger` 与包级日志函数同步；并发运行时不要直接赋值或读取 `DefaultLogger`。不要并发修改正在使用的 `Logger.Level`、`Logger.Outputs` 或其底层切片；自定义输出需自行保证并发安全。
- 同步范围仅覆盖包级日志函数；自定义输出回调在包级日志调用中既不能调用 `SetDefaultLogger`，也不能递归调用包级 `Info` / `Debug` / `Warn` / `Error`。外层调用持有 `RLock`：调用 `SetDefaultLogger` 会等待自身释放读锁；若另一 goroutine 正等待写锁，递归的 `RLock` 也会阻塞，导致死锁（`sync.RWMutex` 不支持递归读锁）。输出回调不要经由任何日志器再次进入自身，避免递归输出，即使没有锁竞争也不安全。
- 输出返回错误时仍继续后续输出，向 stderr 写入固定提示 `logs: output failed; further failures suppressed for one minute`。提示全进程最多每分钟一次，不包含原错误或日志正文；日志方法不返回输出错误。

## 关闭文件输出

`NewFileOutput` 返回 `Output`，该接口不含 `Close`；需要关闭时通过接口断言访问 `FileOutput.Close`：

```go
output := logs.NewFileOutput(logs.Level_Info, "./logs/job.log", 10, 7, 5)
// 使用 output；关闭前先停止并等待所有向它写入的调用结束。
if err := output.(interface{ Close() error }).Close(); err != nil {
    // 由调用方处理关闭错误。
}
```

替换默认日志器不会自动关闭旧输出。`RoutedOutput`（包括 `NewBizFileOutput` 的返回值）不提供 `Close`；若需管理文件生命周期，先创建并保留 `NewFileOutput` 的结果，再用 `IncludeBiz` 包装，停写后关闭原输出。
