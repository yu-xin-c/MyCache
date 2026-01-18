package lru

import (
	"container/heap"
	"container/list"
	"time"
)

// Cache is a LRU cache. It is not safe for concurrent access.
type Cache struct {
	maxBytes   int64
	nbytes     int64
	ll         *list.List
	cache      map[string]*list.Element
	expireHeap *expireHeap
	// optional and executed when an entry is purged.
	OnEvicted func(key string, value Value)
}

type entry struct {
	key      string
	value    Value
	expireAt time.Time
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

type expireItem struct {
	expireAt time.Time
	key      string
}

type expireHeap []expireItem

func (h expireHeap) Len() int           { return len(h) }
func (h expireHeap) Less(i, j int) bool { return h[i].expireAt.Before(h[j].expireAt) }
func (h expireHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *expireHeap) Push(x interface{}) {
	*h = append(*h, x.(expireItem))
}

func (h *expireHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	eh := &expireHeap{}
	heap.Init(eh)
	return &Cache{
		maxBytes:   maxBytes,
		ll:         list.New(),
		cache:      make(map[string]*list.Element),
		expireHeap: eh,
		OnEvicted:  onEvicted,
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value Value, ttl time.Duration) {
	var expireAt time.Time
	if ttl > 0 {
		expireAt = time.Now().Add(ttl)
	}
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
		kv.expireAt = expireAt
		if !expireAt.IsZero() {
			heap.Push(c.expireHeap, expireItem{expireAt, key})
		}
	} else {
		ele := c.ll.PushFront(&entry{key, value, expireAt})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
		if !expireAt.IsZero() {
			heap.Push(c.expireHeap, expireItem{expireAt, key})
		}
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Get look ups a key's value
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		kv := ele.Value.(*entry)
		if !kv.expireAt.IsZero() && time.Now().After(kv.expireAt) {
			c.removeElement(ele)
			return nil, false
		}
		c.ll.MoveToFront(ele)
		return kv.value, true
	}
	return
}

// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

// removeElement removes the given element
func (c *Cache) removeElement(ele *list.Element) {
	c.ll.Remove(ele)
	kv := ele.Value.(*entry)
	delete(c.cache, kv.key)
	c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
	if c.OnEvicted != nil {
		c.OnEvicted(kv.key, kv.value)
	}
}

// CleanExpired removes expired items
func (c *Cache) CleanExpired() {
	now := time.Now()
	for c.expireHeap.Len() > 0 {
		item := (*c.expireHeap)[0]
		if now.After(item.expireAt) {
			heap.Pop(c.expireHeap)
			if ele, ok := c.cache[item.key]; ok {
				c.removeElement(ele)
			}
		} else {
			break
		}
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}
