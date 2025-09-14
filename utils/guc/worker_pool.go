package guc

import (
	"context"
	"github.com/jdkhome/gdk/errs"
	"github.com/jdkhome/gdk/logs"
	"runtime/debug"
	"sync"
)

// Task 表示一个需要执行的任务
type Task struct {
	Ctx context.Context // 任务上下文，用于传递超时、元数据等
	Biz []string        // 业务标识，用于日志等场景
	Fn  func()          // 任务执行函数
}

// WorkerPool 协程池
type WorkerPool struct {
	poolSize int            // 协程池大小（工作协程数量）
	tasks    chan *Task     // 任务队列（缓冲大小等于poolSize）
	quit     chan struct{}  // 退出信号通道
	wg       sync.WaitGroup // 用于等待所有工作协程退出
}

// NewWorkerPool 创建新的协程池
// size: 工作协程的数量（同时运行的最大任务数）
func NewWorkerPool(size int) *WorkerPool {

	pool := &WorkerPool{
		poolSize: size,
		tasks:    make(chan *Task, size), // 缓冲大小等于协程数，满时提交会阻塞
		quit:     make(chan struct{}),
	}

	pool.start()

	return pool
}

// start 启动协程池，创建并启动工作协程
func (p *WorkerPool) start() {
	for i := 0; i < p.poolSize; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done() // 工作协程退出时通知WaitGroup
			for {
				select {
				case task, ok := <-p.tasks:
					if !ok {
						// 任务通道已关闭，退出工作协程
						return
					}
					p.exec(task.Ctx, task.Biz, task.Fn)
				case <-p.quit:
					return
				}
			}
		}(i)
	}
}

// exec 执行单个任务，捕获并处理任务中的panic
func (p *WorkerPool) exec(ctx context.Context, biz []string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			// 记录panic日志（包含堆栈信息）
			logs.Error(ctx, biz, "WorkerPool task panic! err:%v stack:%s", r, debug.Stack())
		}
	}()
	fn() // 执行任务函数
}

// Shutdown 优雅关闭协程池（修改后）
func (p *WorkerPool) Shutdown() {
	close(p.tasks)
	close(p.quit) // 发送退出信号
	p.wg.Wait()   // 等待所有工作协程退出
}

// Submit 提交任务
func (p *WorkerPool) Submit(task *Task) error {
	biz := []string{"WorkerPool", "Submit"}
	select {
	case p.tasks <- task:
		return nil
	case <-p.quit:
		return errs.NewBizErr(biz, "协程池已关闭")
	}
}
