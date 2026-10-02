package logs

import (
	"context"
	"sync"
)

var (
	Console       = NewConsoleOutput(Level_Info)
	AppDefaultLog = NewFileOutput(Level_Info, "./logs/app_default.log", 10, 7, 5)
	CommonError   = NewFileOutput(Level_Error, "./logs/common_error.log", 10, 7, 5)
	DefaultLogger = NewLogger(Level_Info, []Output{Console, AppDefaultLog, CommonError})
	defaultMu     sync.RWMutex
)

// SetDefaultLogger 与包级日志函数同步；并发运行时不要直接赋值 DefaultLogger。
func SetDefaultLogger(logger *Logger) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	DefaultLogger = logger
}

func Info(ctx context.Context, biz []string, msg string, args ...any) {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	DefaultLogger.Info(ctx, biz, msg, args...)
}
func Debug(ctx context.Context, biz []string, msg string, args ...any) {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	DefaultLogger.Debug(ctx, biz, msg, args...)
}
func Warn(ctx context.Context, biz []string, msg string, args ...any) {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	DefaultLogger.Warn(ctx, biz, msg, args...)
}
func Error(ctx context.Context, biz []string, msg string, args ...any) {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	DefaultLogger.Error(ctx, biz, msg, args...)
}
