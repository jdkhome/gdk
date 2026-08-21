package random_util

import (
	cache "github.com/Code-Hex/go-generics-cache"
	"testing"
)

func TestRandomGetFromCache_Empty(t *testing.T) {
	c := cache.New[string, int]()
	if got := RandomGetFromCache(c, 3); got != nil {
		t.Errorf("空缓存应返回 nil, got %v", got)
	}
}

func TestRandomGetFromCache_NonPositiveN(t *testing.T) {
	c := cache.New[string, int]()
	c.Set("a", 1)

	if got := RandomGetFromCache(c, 0); got != nil {
		t.Errorf("n=0 应返回 nil, got %v", got)
	}
	if got := RandomGetFromCache(c, -1); got != nil {
		t.Errorf("n=-1 应返回 nil, got %v", got)
	}
}

func TestRandomGetFromCache_NGreaterThanSize(t *testing.T) {
	c := cache.New[string, int]()
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	got := RandomGetFromCache(c, 10)
	if len(got) != 3 {
		t.Errorf("len = %d, want 3", len(got))
	}
}

func TestRandomGetFromCache_Normal(t *testing.T) {
	c := cache.New[string, int]()
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	got := RandomGetFromCache(c, 2)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}

	seen := map[int]bool{1: true, 2: true, 3: true}
	for _, v := range got {
		if !seen[v] {
			t.Errorf("返回了缓存中不存在的值 %v", v)
		}
	}
}
