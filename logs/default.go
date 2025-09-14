package logs

import "context"

var (
	Console       = NewConsoleOutput(Level_Info)
	AppDefaultLog = NewFileOutput(Level_Info, "./logs/app_default.log", 10, 7, 5)
	CommonError   = NewFileOutput(Level_Error, "./logs/common_error.log", 10, 7, 5)
)

var (
	DefaultLogger = NewLogger(Level_Info, []Output{Console, AppDefaultLog, CommonError})
)

func Info(ctx context.Context, biz []string, msg string, args ...any) {
	DefaultLogger.Info(ctx, biz, msg, args...)
}
func Debug(ctx context.Context, biz []string, msg string, args ...any) {
	DefaultLogger.Debug(ctx, biz, msg, args...)
}
func Warn(ctx context.Context, biz []string, msg string, args ...any) {
	DefaultLogger.Warn(ctx, biz, msg, args...)
}
func Error(ctx context.Context, biz []string, msg string, args ...any) {
	DefaultLogger.Error(ctx, biz, msg, args...)
}
