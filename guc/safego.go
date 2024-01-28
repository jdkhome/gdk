package guc

import (
	"context"
	"fmt"
	"gdk/exception"
)

func GoRunnable(ctx context.Context, fun func(ctx context.Context), catchList ...exception.Catch) {
	// 没传catch时 给个兜底
	if len(catchList) == 0 {
		catchList = []exception.Catch{
			{exception.Biz, func(err any) {
				fmt.Printf("捕获到%v异常: %v\n", exception.Biz, err)
			}},
			{exception.Err, func(err any) {
				fmt.Printf("捕获到%v异常: %v\n", exception.Err, err)
			}},
			{exception.Any, func(err any) {
				fmt.Printf("捕获到%v异常: %v\n", exception.Any, err)
			}},
		}
	}

	go func() {
		exception.TryCatch(func() {
			// 执行实际的逻辑
			fun(ctx)
		},
			catchList...,
		)
	}()
}

func GoCallable[T any](ctx context.Context, fun func(ctx context.Context) T, catchList ...exception.Catch) *Future[T] {
	// 没传catch时 给个兜底
	if len(catchList) == 0 {
		catchList = []exception.Catch{
			{exception.Biz, func(err any) {
				fmt.Printf("捕获到%v异常: %v\n", exception.Biz, err)
			}},
			{exception.Err, func(err any) {
				fmt.Printf("捕获到%v异常: %v\n", exception.Err, err)
			}},
			{exception.Any, func(err any) {
				fmt.Printf("捕获到%v异常: %v\n", exception.Any, err)
			}},
		}
	}

	future := newFuture[T]()
	future.wg.Add(1)
	go func() {
		defer future.wg.Done()
		exception.TryCatch(func() {
			// 执行实际的逻辑
			result := fun(ctx)
			future.setResult(&result)
		},
			catchList...,
		)
	}()

	return future
}
