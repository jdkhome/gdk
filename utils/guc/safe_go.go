package guc

import (
	"context"
	"github.com/jdkhome/gdk/logs"
	"runtime/debug"
)

// SafeGo 安全启动协程的函数，防止panic影响主进程
func SafeGo(ctx context.Context, biz []string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logs.Error(ctx, biz, "SafeGo Panic! err:%v stack:%s", r, debug.Stack())
			}
		}()

		// 执行传入的函数
		fn()
	}()
}
