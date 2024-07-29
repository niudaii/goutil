package history

import "sync"

// Tracker 用于跟踪最近的查询记录
type Tracker struct {
	history    []string
	historySet map[string]struct{} // 使用 map 来快速检查重复项
	limit      int
	mu         sync.Mutex
}

// NewHistoryTracker 创建一个新的 HistoryTracker
func NewHistoryTracker(limit int) *Tracker {
	return &Tracker{
		history:    make([]string, 0, limit),
		historySet: make(map[string]struct{}),
		limit:      limit,
	}
}

// AddRecord 添加一条新的记录
func (ht *Tracker) AddRecord(record string) {
	if record == "" {
		return
	}

	ht.mu.Lock() // 加锁保证线程安全
	defer ht.mu.Unlock()

	// 检查记录是否已经存在
	if _, exists := ht.historySet[record]; exists {
		return
	}

	// 如果达到限制，移除最后一个元素
	if len(ht.history) >= ht.limit {
		lastRecord := ht.history[len(ht.history)-1]
		delete(ht.historySet, lastRecord) // 从 set 中也移除
		ht.history = ht.history[:ht.limit-1]
	}
	// 将新记录插入到切片的开头
	ht.history = append([]string{record}, ht.history...)
	ht.historySet[record] = struct{}{} // 添加到 set 中标记已存在
}

// GetHistory 返回当前的查询记录
func (ht *Tracker) GetHistory() []string {
	ht.mu.Lock() // 加锁以避免读取时被修改
	defer ht.mu.Unlock()

	// 返回一个历史记录的副本以避免外部修改
	historyCopy := make([]string, len(ht.history))
	copy(historyCopy, ht.history)
	return historyCopy
}

// ClearHistory 清除所有历史记录
func (ht *Tracker) ClearHistory() {
	ht.mu.Lock() // 加锁以确保线程安全
	defer ht.mu.Unlock()

	// 重置历史记录切片
	ht.history = make([]string, 0, ht.limit)
	ht.historySet = make(map[string]struct{})
}
