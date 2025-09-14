package random_util

import (
	cache "github.com/Code-Hex/go-generics-cache"
	"math/rand"
	"time"
)

// RandomGetFromCache 封装函数，用于从缓存中随机获取 n 条数据
func RandomGetFromCache[K comparable, V any](cache *cache.Cache[K, V], n int) []V {
	keys := cache.Keys()
	numKeys := len(keys)

	if numKeys == 0 || n <= 0 {
		return nil
	}

	if n > numKeys {
		n = numKeys
	}

	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())

	// 打乱键的顺序
	rand.Shuffle(numKeys, func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})

	// 选取前 n 个键
	selectedKeys := keys[:n]
	result := make([]V, 0, n)

	// 根据选取的键获取对应的值
	for _, key := range selectedKeys {
		if val, ok := cache.Get(key); ok {
			result = append(result, val)
		}
	}

	return result
}
