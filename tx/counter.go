package tx

import (
	"errors"
	"sync"
)

// GlobalCounter 表示全局循环自增计数器
type GlobalCounter struct {
	mu       sync.Mutex
	counter  uint64
	minValue uint64
	maxValue uint64
}

// 全局计数器实例
var (
	defaultCounter = NewGlobalCounter(1, 1000) // 默认范围从1到1000
)

// NewGlobalCounter 创建一个新的循环自增计数器，指定最小值和最大值
func NewGlobalCounter(min, max uint64) *GlobalCounter {
	if min >= max {
		panic("最小值必须小于最大值")
	}
	return &GlobalCounter{
		counter:  min - 1, // 初始值设为最小值减1，以便首次调用Next返回最小值
		minValue: min,
		maxValue: max,
	}
}

// Next 返回下一个自增值（线程安全）
func (c *GlobalCounter) Next() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.counter++
	if c.counter > c.maxValue {
		c.counter = c.minValue // 回绕到最小值
	}
	return c.counter
}

// NextDefault 返回默认全局计数器的下一个值
func NextDefault() uint64 {
	return defaultCounter.Next()
}

// Reset 将计数器重置为最小值
func (c *GlobalCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter = c.minValue - 1
}

// Current 返回当前计数器值（不增加）
func (c *GlobalCounter) Current() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counter
}

// Set 设置计数器的当前值
func (c *GlobalCounter) Set(value uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if value < c.minValue || value > c.maxValue {
		return errors.New("设置的值超出范围")
	}
	c.counter = value
	return nil
}
