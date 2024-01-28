package guc

import "sync"

type Future[T any] struct {
	result *T
	wg     sync.WaitGroup
}

func newFuture[T any]() *Future[T] {
	return &Future[T]{wg: sync.WaitGroup{}}
}

func (f *Future[T]) setResult(result *T) {
	f.result = result
}

func (f *Future[T]) GetResult() T {
	f.wg.Wait()
	return *f.result
}
