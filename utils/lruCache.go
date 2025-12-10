package utils

import (
	"container/list"
	"sync"
)

// LRUCache LRU缓存实现，线程安全
type LRUCache struct {
	capacity int
	cache    map[string]*list.Element
	lru      *list.List
	mu       sync.RWMutex
}

// lruEntry 缓存条目
type lruEntry struct {
	key   string
	value interface{}
}

// NewLRUCache 创建新的LRU缓存
func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = 100 // 默认容量
	}
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

// Get 获取缓存项，如果存在则返回值，否则返回nil
func (c *LRUCache) Get(key string) interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.cache[key]; exists {
		// 移到最前面（最近使用）
		c.lru.MoveToFront(elem)
		return elem.Value.(*lruEntry).value
	}
	return nil
}

// Set 设置缓存项
func (c *LRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果已存在，更新值并移到最前面
	if elem, exists := c.cache[key]; exists {
		c.lru.MoveToFront(elem)
		elem.Value.(*lruEntry).value = value
		return
	}

	// 新增条目
	entry := &lruEntry{key: key, value: value}
	elem := c.lru.PushFront(entry)
	c.cache[key] = elem

	// 检查是否超过容量，删除最久未使用的
	if c.lru.Len() > c.capacity {
		oldest := c.lru.Back()
		if oldest != nil {
			c.lru.Remove(oldest)
			delete(c.cache, oldest.Value.(*lruEntry).key)
		}
	}
}

// Delete 删除指定key的缓存项
func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.cache[key]; exists {
		c.lru.Remove(elem)
		delete(c.cache, key)
	}
}

// Clear 清空所有缓存
func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*list.Element)
	c.lru.Init()
}

// Len 返回当前缓存项数量
func (c *LRUCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lru.Len()
}

// Capacity 返回缓存容量
func (c *LRUCache) Capacity() int {
	return c.capacity
}
