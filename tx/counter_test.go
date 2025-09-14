package tx

import (
	"sync"
	"testing"
)

func TestGlobalCounter_Next(t *testing.T) {
	min, max := uint64(1), uint64(10)
	c := NewGlobalCounter(min, max)

	// 测试基本递增
	for i := min; i <= max; i++ {
		if val := c.Next(); val != i {
			t.Errorf("Next() = %d, want %d", val, i)
		}
	}

	// 测试循环回绕
	if val := c.Next(); val != min {
		t.Errorf("After wrap around, Next() = %d, want %d", val, min)
	}
}

func TestGlobalCounter_ConcurrentNext(t *testing.T) {
	min, max := uint64(1), uint64(100)
	c := NewGlobalCounter(min, max)
	results := make(map[uint64]int)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 启动多个goroutine并发调用Next()
	workers := 10
	iterations := 20
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				val := c.Next()
				mu.Lock()
				results[val]++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	// 验证结果：每个值应该出现workers/周期次
	expectedCount := workers * iterations / int(max-min+1)
	remainder := workers * iterations % int(max-min+1)

	for k, v := range results {
		// 对于min~min+remainder-1的值，会多出现一次
		if k >= min && k < min+uint64(remainder) {
			if v != expectedCount+1 {
				t.Errorf("Value %d count = %d, want %d", k, v, expectedCount+1)
			}
		} else if v != expectedCount {
			t.Errorf("Value %d count = %d, want %d", k, v, expectedCount)
		}
	}
}

func TestGlobalCounter_Reset(t *testing.T) {
	min, max := uint64(5), uint64(15)
	c := NewGlobalCounter(min, max)

	// 先递增几次
	for i := 0; i < 3; i++ {
		c.Next()
	}

	// 重置
	c.Reset()

	// 验证下一个值是min
	if val := c.Next(); val != min {
		t.Errorf("After Reset, Next() = %d, want %d", val, min)
	}
}

func TestGlobalCounter_Current(t *testing.T) {
	min, max := uint64(100), uint64(200)
	c := NewGlobalCounter(min, max)

	// 初始值应为min-1
	if val := c.Current(); val != min-1 {
		t.Errorf("Initial Current() = %d, want %d", val, min-1)
	}

	// 递增后验证
	c.Next() // 现在应该是min
	if val := c.Current(); val != min {
		t.Errorf("After one Next(), Current() = %d, want %d", val, min)
	}
}

func TestGlobalCounter_Set(t *testing.T) {
	min, max := uint64(10), uint64(20)
	c := NewGlobalCounter(min, max)

	// 设置有效值
	testVal := uint64(15)
	if err := c.Set(testVal); err != nil {
		t.Errorf("Set(%d) error = %v, want nil", testVal, err)
	}

	// 验证当前值
	if val := c.Current(); val != testVal {
		t.Errorf("After Set(%d), Current() = %d, want %d", testVal, val, testVal)
	}

	// 设置超出范围的值
	if err := c.Set(min - 1); err == nil {
		t.Errorf("Set(%d) error = %v, want '设置的值超出范围'", min-1, err)
	}

	if err := c.Set(max + 1); err == nil {
		t.Errorf("Set(%d) error = %v, want '设置的值超出范围'", max+1, err)
	}
}

func TestNextDefault(t *testing.T) {
	// 测试默认计数器
	first := NextDefault()
	second := NextDefault()

	if second != first+1 {
		t.Errorf("NextDefault() sequence incorrect: first=%d, second=%d", first, second)
	}
}
