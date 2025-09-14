package tx

import (
	"sync"
	"time"
)

// TransactionManager 事务管理器
type TransactionManager[T any] struct {
	maxID       uint64
	nextID      uint64
	lock        sync.Mutex
	transaction sync.Map
}

// NewTransactionManager 创建新的事务管理器
func NewTransactionManager[T any](maxID uint64) *TransactionManager[T] {
	return &TransactionManager[T]{
		maxID:  maxID,
		nextID: 0,
	}
}

// CreateTransaction 创建新事务
func (ta *TransactionManager[T]) CreateTransaction(data T) uint64 {
	for {
		ta.lock.Lock()
		// 尝试获取一个ID
		id := ta.nextID
		ta.nextID = (ta.nextID + 1) % (ta.maxID + 1)
		ta.lock.Unlock()

		// 检查ID是否已被使用
		if _, loaded := ta.transaction.LoadOrStore(id, data); !loaded {
			return id
		}

		// 如果ID已被使用，则等待一段时间再试
		time.Sleep(10 * time.Microsecond)
	}
}

// GetTransaction 获取事务
func (ta *TransactionManager[T]) GetTransaction(id uint64) (T, bool) {
	data, exists := ta.transaction.Load(id)
	var zero T
	if !exists {
		return zero, false
	}
	return data.(T), true
}

// ReleaseTransaction 释放事务
func (ta *TransactionManager[T]) ReleaseTransaction(id uint64) {
	ta.transaction.Delete(id)
}

// CleanupExpiredTransactions 清理过期事务
func (ta *TransactionManager[T]) CleanupExpiredTransactions(isExpired func(T) bool) {
	ta.transaction.Range(func(key, value interface{}) bool {
		id := key.(uint64)
		data := value.(T)
		if isExpired(data) {
			ta.transaction.Delete(id)
		}
		return true
	})
}
