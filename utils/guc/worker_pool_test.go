package guc

import (
	"context"
	"github.com/jdkhome/gdk/errs"
	"github.com/jdkhome/gdk/traces"
	"sync"
	"testing"
	"time"
)

// 测试正常提交任务并执行
func TestWorkerPool_SubmitNormal(t *testing.T) {
	poolSize := 5
	pool := NewWorkerPool(poolSize)
	defer pool.Shutdown()

	var wg sync.WaitGroup
	taskCount := 10
	result := make([]int, taskCount)

	// 提交多个任务
	for i := 0; i < taskCount; i++ {
		wg.Add(1)
		idx := i
		err := pool.Submit(&Task{
			Ctx: traces.NewCtx(),
			Biz: []string{"test", "normal"},
			Fn: func() {
				defer wg.Done()
				result[idx] = idx                 // 标记任务已执行
				time.Sleep(10 * time.Millisecond) // 模拟耗时操作
			},
		})

		if err != nil {
			t.Errorf("提交任务 %d 失败: %v", i, err)
		}
	}

	// 等待所有任务完成
	wg.Wait()

	// 验证所有任务都已执行
	for i := 0; i < taskCount; i++ {
		if result[i] != i {
			t.Errorf("任务 %d 未正确执行，结果为 %d", i, result[i])
		}
	}
}

// 测试协程池关闭后无法提交任务
func TestWorkerPool_Shutdown(t *testing.T) {
	pool := NewWorkerPool(3)
	pool.Shutdown() // 立即关闭

	// 尝试提交任务
	err := pool.Submit(&Task{
		Ctx: traces.NewCtx(),
		Fn:  func() {},
	})

	if err == nil {
		t.Error("预期提交任务失败，但成功了")
		return
	}

	if !errs.Is(err, errs.BizErr) {
		t.Errorf("预期错误类型为 *errs.BizErr，但实际为 %T", err)
		return
	}
}

// 测试任务执行时发生panic的情况
func TestWorkerPool_TaskPanic(t *testing.T) {
	pool := NewWorkerPool(1)
	defer pool.Shutdown()

	var wg sync.WaitGroup
	wg.Add(1)

	// 提交一个会panic的任务
	err := pool.Submit(&Task{
		Ctx: traces.NewCtx(),
		Biz: []string{"test", "panic"},
		Fn: func() {
			defer wg.Done()
			panic("测试panic")
		},
	})

	if err != nil {
		t.Fatalf("提交任务失败: %v", err)
	}

	// 等待任务执行完成（包括panic处理）
	wg.Wait()

	// 验证协程池仍然可以正常工作
	err = pool.Submit(&Task{
		Ctx: traces.NewCtx(),
		Fn:  func() {},
	})

	if err != nil {
		t.Error("panic后协程池无法接受新任务")
	}
}

// 测试任务队列满时的阻塞行为
func TestWorkerPool_TaskQueueFull(t *testing.T) {
	poolSize := 2
	pool := NewWorkerPool(poolSize)
	defer pool.Shutdown()

	// 使用WaitGroup确保所有忙碌任务开始执行
	var busyWg sync.WaitGroup
	busyWg.Add(poolSize)

	// 提交poolSize个任务，填满工作协程
	for i := 0; i < poolSize; i++ {
		idx := i
		err := pool.Submit(&Task{
			Fn: func() {
				busyWg.Done() // 通知测试主线程该任务已开始执行
				t.Logf("busy start %d", idx)
				time.Sleep(3000 * time.Millisecond) // 保持工作协程忙碌
				t.Logf("busy end %d", idx)
			},
		})
		if err != nil {
			t.Fatalf("提交忙碌任务失败: %v", err)
		}
	}

	// 等待所有工作协程都开始处理任务
	busyWg.Wait()

	// 此时工作协程都在忙碌，任务队列应该为空
	// 再提交poolSize个任务，填满任务队列
	for i := 0; i < poolSize; i++ {
		idx := i + poolSize
		err := pool.Submit(&Task{
			Fn: func() {
				t.Logf("queue start %d", idx)
				time.Sleep(100 * time.Millisecond)
				t.Logf("queue end %d", idx)
			},
		})
		if err != nil {
			t.Fatalf("提交队列任务失败: %v", err)
		}
	}

	// 此时任务队列应该已满，再提交任务会阻塞
	startTime := time.Now()
	ch := make(chan error, 1)

	// 在单独的goroutine中提交任务，避免阻塞测试主线程
	go func() {
		err := pool.Submit(&Task{
			Fn: func() {
				t.Logf("new start")
			},
		})
		ch <- err
	}()

	// 验证提交操作会阻塞一段时间（直到有工作协程空闲）
	select {
	case err := <-ch:
		if err != nil {
			t.Errorf("提交任务失败: %v", err)
		}
		duration := time.Since(startTime)
		if duration < 50*time.Millisecond {
			t.Errorf("任务提交没有正确阻塞，阻塞时间过短: %v", duration)
		}
	case <-time.After(4000 * time.Millisecond): // 等待时间应大于任务执行时间
		t.Error("任务提交阻塞时间过长，可能出现死锁")
	}
}

// 测试上下文取消对任务的影响
func TestWorkerPool_TaskContextCancel(t *testing.T) {
	pool := NewWorkerPool(1)
	defer pool.Shutdown()

	ctx, cancel := context.WithCancel(traces.NewCtx())
	var wg sync.WaitGroup
	wg.Add(1)

	// 提交一个会检查上下文的任务
	err := pool.Submit(&Task{
		Ctx: ctx,
		Fn: func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				// 预期会执行到这里
			case <-time.After(100 * time.Millisecond):
				t.Error("任务没有响应上下文取消")
			}
		},
	})

	if err != nil {
		t.Fatalf("提交任务失败: %v", err)
	}

	// 立即取消上下文
	cancel()
	wg.Wait()
}

// 测试大规模任务并发执行
func TestWorkerPool_ConcurrentTasks(t *testing.T) {
	poolSize := 10
	taskCount := 1000
	pool := NewWorkerPool(poolSize)
	defer pool.Shutdown()

	var wg sync.WaitGroup
	wg.Add(taskCount)
	counter := 0
	var mu sync.Mutex

	// 提交大量任务
	for i := 0; i < taskCount; i++ {
		err := pool.Submit(&Task{
			Fn: func() {
				defer wg.Done()
				mu.Lock()
				counter++
				mu.Unlock()
			},
		})
		if err != nil {
			t.Errorf("提交任务 %d 失败: %v", i, err)
		}
	}

	// 等待所有任务完成
	wg.Wait()

	// 验证所有任务都已执行
	if counter != taskCount {
		t.Errorf("任务执行计数不正确，预期 %d，实际 %d", taskCount, counter)
	}
}
