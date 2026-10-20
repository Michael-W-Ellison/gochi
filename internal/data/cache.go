package data

import (
	"container/list"
	"sync"
	"time"
)

// Cache is a least-recently-used cache bounded by the total size of its
// entries. Entries can also expire after a time to live.
type Cache struct {
	mu         sync.Mutex
	maxSize    int64 // Total size of entries allowed, in bytes
	size       int64
	defaultTTL time.Duration            // 0 means entries don't expire
	entries    map[string]*list.Element // Of *cacheEntry
	lru        *list.List               // Most recently used first
	now        func() time.Time

	hits, misses, evictions int64
}

// cacheEntry is one cached value
type cacheEntry struct {
	key       string
	value     interface{}
	size      int64
	expiresAt time.Time // Zero if it doesn't expire
}

// CacheStats describes how a cache is being used
type CacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64 // Entries removed to make room
	Size      int64 // Total size of entries, in bytes
	MaxSize   int64
	Entries   int
}

// HitRate returns the share of lookups that found an entry (0.0 to 1.0)
func (s CacheStats) HitRate() float64 {
	if total := s.Hits + s.Misses; total > 0 {
		return float64(s.Hits) / float64(total)
	}
	return 0
}

// NewCache creates a cache holding up to maxSize bytes of entries, which
// expire after defaultTTL (0 for never)
func NewCache(maxSize int64, defaultTTL time.Duration) *Cache {
	return &Cache{
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		entries:    make(map[string]*list.Element),
		lru:        list.New(),
		now:        time.Now,
	}
}

// Set stores a value of the given size under key, replacing any value
// there, and evicts the least recently used entries if the cache is over
// its size. A value larger than the whole cache is not stored.
func (c *Cache) Set(key string, value interface{}, size int64) {
	c.SetWithTTL(key, value, size, c.defaultTTL)
}

// SetWithTTL stores a value that expires after ttl (0 for never)
func (c *Cache) SetWithTTL(key string, value interface{}, size int64, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.entries[key]; ok {
		c.removeLocked(element)
	}
	if size > c.maxSize {
		return
	}
	entry := &cacheEntry{key: key, value: value, size: size}
	if ttl > 0 {
		entry.expiresAt = c.now().Add(ttl)
	}
	c.entries[key] = c.lru.PushFront(entry)
	c.size += size

	for c.size > c.maxSize {
		c.removeLocked(c.lru.Back())
		c.evictions++
	}
}

// Get returns the value stored under key, if it is there and has not
// expired, and marks it as recently used
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.entries[key]
	if !ok {
		c.misses++
		return nil, false
	}
	entry := element.Value.(*cacheEntry)
	if c.expiredLocked(entry) {
		c.removeLocked(element)
		c.misses++
		return nil, false
	}
	c.lru.MoveToFront(element)
	c.hits++
	return entry.value, true
}

// Delete removes the value stored under key
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if element, ok := c.entries[key]; ok {
		c.removeLocked(element)
	}
}

// Clear removes every entry
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*list.Element)
	c.lru.Init()
	c.size = 0
}

// Cleanup removes expired entries and returns how many it removed
func (c *Cache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for element := c.lru.Back(); element != nil; {
		prev := element.Prev()
		if c.expiredLocked(element.Value.(*cacheEntry)) {
			c.removeLocked(element)
			removed++
		}
		element = prev
	}
	return removed
}

// Stats returns the cache's statistics
func (c *Cache) Stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CacheStats{Hits: c.hits, Misses: c.misses, Evictions: c.evictions,
		Size: c.size, MaxSize: c.maxSize, Entries: c.lru.Len()}
}

// expiredLocked reports whether an entry has expired. Caller must hold
// c.mu.
func (c *Cache) expiredLocked(entry *cacheEntry) bool {
	return !entry.expiresAt.IsZero() && !c.now().Before(entry.expiresAt)
}

// removeLocked removes an entry. Caller must hold c.mu.
func (c *Cache) removeLocked(element *list.Element) {
	entry := element.Value.(*cacheEntry)
	c.lru.Remove(element)
	delete(c.entries, entry.key)
	c.size -= entry.size
}
