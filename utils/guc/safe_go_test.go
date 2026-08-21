package guc

import (
	"context"
	"testing"
	"time"
)

func TestSafeGo_Normal(t *testing.T) {
	done := make(chan struct{})
	SafeGo(context.Background(), []string{"test"}, func() {
		close(done)
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("SafeGo 未执行传入函数")
	}
}

func TestSafeGo_Panic(t *testing.T) {
	// panic 应被捕获，不影响主进程
	SafeGo(context.Background(), []string{"test"}, func() {
		panic("boom")
	})

	// 等待 panic 处理完成
	time.Sleep(10 * time.Millisecond)

	// 若 panic 未捕获，测试进程早已崩溃，能走到这里说明正常
}
