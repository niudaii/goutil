package memkv

import (
	"sync"
	"time"
)

// 在main函数或初始化函数中调用
func init() {
	// 启动定期清理过期缓存
	EntryCache.StartCleanup()
}

// MemoryCache **新增：内存缓存结构**
type MemoryCache struct {
	data sync.Map // key: string, value: time.Time
	mu   sync.RWMutex
}

var (
	// 全局内存缓存实例
	EntryCache = &MemoryCache{}
)

// Set **新增：缓存操作方法**
func (mc *MemoryCache) Set(key string) {
	mc.data.Store(key, time.Now())
}

func (mc *MemoryCache) Exists(key string) bool {
	value, exists := mc.data.Load(key)
	if !exists {
		return false
	}

	// 检查是否超过24小时
	if time.Since(value.(time.Time)) > 24*time.Hour {
		mc.data.Delete(key) // 清理过期数据
		return false
	}
	return true
}

// StartCleanup **新增：定期清理过期缓存**
func (mc *MemoryCache) StartCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour) // 每小时清理一次
		defer ticker.Stop()

		for range ticker.C {
			mc.data.Range(func(key, value interface{}) bool {
				if time.Since(value.(time.Time)) > 24*time.Hour {
					mc.data.Delete(key)
				}
				return true
			})
		}
	}()
}
