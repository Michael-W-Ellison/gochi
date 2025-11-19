package data

import (
	"container/list"
	"sync"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Key        string
	Data       interface{}
	Size       int64
	AccessTime time.Time
	CreateTime time.Time
	TTL        time.Duration
	AccessCount int
}

// CacheStats provides statistics about cache performance
type CacheStats struct {
	Hits        int64
	Misses      int64
	Evictions   int64
	CurrentSize int64
	MaxSize     int64
	EntryCount  int
}

// DataCache implements an LRU cache with TTL support
type DataCache struct {
	mu sync.RWMutex

	maxSize      int64 // Maximum cache size in bytes
	currentSize  int64
	entries      map[string]*list.Element
	lruList      *list.List
	defaultTTL   time.Duration

	// Statistics
	hits      int64
	misses    int64
	evictions int64
}

// cacheItem is stored in the LRU list
type cacheItem struct {
	key   string
	entry *CacheEntry
}

// NewDataCache creates a new cache with specified max size
func NewDataCache(maxSizeBytes int64, defaultTTL time.Duration) *DataCache {
	return &DataCache{
		maxSize:    maxSizeBytes,
		entries:    make(map[string]*list.Element),
		lruList:    list.New(),
		defaultTTL: defaultTTL,
	}
}

// Set adds or updates an item in the cache
func (c *DataCache) Set(key string, data interface{}, size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if entry already exists
	if element, exists := c.entries[key]; exists {
		// Update existing entry
		item := element.Value.(*cacheItem)
		oldSize := item.entry.Size

		item.entry.Data = data
		item.entry.Size = size
		item.entry.AccessTime = time.Now()
		item.entry.AccessCount++

		c.currentSize += (size - oldSize)
		c.lruList.MoveToFront(element)
	} else {
		// Create new entry
		entry := &CacheEntry{
			Key:         key,
			Data:        data,
			Size:        size,
			AccessTime:  time.Now(),
			CreateTime:  time.Now(),
			TTL:         c.defaultTTL,
			AccessCount: 1,
		}

		item := &cacheItem{
			key:   key,
			entry: entry,
		}

		element := c.lruList.PushFront(item)
		c.entries[key] = element
		c.currentSize += size
	}

	// Evict if necessary
	c.evictIfNeeded()
}

// SetWithTTL sets an item with a custom TTL
func (c *DataCache) SetWithTTL(key string, data interface{}, size int64, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &CacheEntry{
		Key:         key,
		Data:        data,
		Size:        size,
		AccessTime:  time.Now(),
		CreateTime:  time.Now(),
		TTL:         ttl,
		AccessCount: 1,
	}

	item := &cacheItem{
		key:   key,
		entry: entry,
	}

	if element, exists := c.entries[key]; exists {
		c.currentSize -= element.Value.(*cacheItem).entry.Size
		c.lruList.Remove(element)
	}

	element := c.lruList.PushFront(item)
	c.entries[key] = element
	c.currentSize += size

	c.evictIfNeeded()
}

// Get retrieves an item from the cache
func (c *DataCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, exists := c.entries[key]
	if !exists {
		c.misses++
		return nil, false
	}

	item := element.Value.(*cacheItem)

	// Check TTL
	if item.entry.TTL > 0 {
		if time.Since(item.entry.CreateTime) > item.entry.TTL {
			// Expired
			c.removeElement(element)
			c.misses++
			return nil, false
		}
	}

	// Update access time and move to front (LRU)
	item.entry.AccessTime = time.Now()
	item.entry.AccessCount++
	c.lruList.MoveToFront(element)
	c.hits++

	return item.entry.Data, true
}

// Delete removes an item from the cache
func (c *DataCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, exists := c.entries[key]; exists {
		c.removeElement(element)
	}
}

// Clear removes all items from the cache
func (c *DataCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*list.Element)
	c.lruList = list.New()
	c.currentSize = 0
}

// Cleanup removes expired entries
func (c *DataCache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for element := c.lruList.Back(); element != nil; {
		item := element.Value.(*cacheItem)
		prev := element.Prev()

		if item.entry.TTL > 0 && time.Since(item.entry.CreateTime) > item.entry.TTL {
			c.removeElement(element)
			removed++
		}

		element = prev
	}

	return removed
}

// GetStats returns cache statistics
func (c *DataCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Hits:        c.hits,
		Misses:      c.misses,
		Evictions:   c.evictions,
		CurrentSize: c.currentSize,
		MaxSize:     c.maxSize,
		EntryCount:  c.lruList.Len(),
	}
}

// GetHitRate returns the cache hit rate (0-1)
func (c *DataCache) GetHitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.hits + c.misses
	if total == 0 {
		return 0.0
	}

	return float64(c.hits) / float64(total)
}

// Helper methods

func (c *DataCache) evictIfNeeded() {
	// Evict least recently used items until under max size
	for c.currentSize > c.maxSize && c.lruList.Len() > 0 {
		element := c.lruList.Back()
		if element != nil {
			c.removeElement(element)
			c.evictions++
		}
	}
}

func (c *DataCache) removeElement(element *list.Element) {
	if element == nil {
		return
	}

	item := element.Value.(*cacheItem)
	c.lruList.Remove(element)
	delete(c.entries, item.key)
	c.currentSize -= item.entry.Size
}

// PetCache is a specialized cache for pet data
type PetCache struct {
	cache *DataCache
}

// NewPetCache creates a new pet-specific cache
func NewPetCache(maxSizeMB int, defaultTTLMinutes int) *PetCache {
	maxSizeBytes := int64(maxSizeMB) * 1024 * 1024
	defaultTTL := time.Duration(defaultTTLMinutes) * time.Minute

	return &PetCache{
		cache: NewDataCache(maxSizeBytes, defaultTTL),
	}
}

// CachePet stores a pet in the cache
func (pc *PetCache) CachePet(petID types.PetID, petData interface{}, sizeBytes int64) {
	pc.cache.Set(string(petID), petData, sizeBytes)
}

// GetPet retrieves a pet from the cache
func (pc *PetCache) GetPet(petID types.PetID) (interface{}, bool) {
	return pc.cache.Get(string(petID))
}

// InvalidatePet removes a pet from the cache
func (pc *PetCache) InvalidatePet(petID types.PetID) {
	pc.cache.Delete(string(petID))
}

// GetStats returns cache statistics
func (pc *PetCache) GetStats() CacheStats {
	return pc.cache.GetStats()
}

// CleanupExpired removes expired entries
func (pc *PetCache) CleanupExpired() int {
	return pc.cache.Cleanup()
}
