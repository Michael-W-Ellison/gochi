package data

import (
	"testing"
	"time"

	"github.com/Michael-W-Ellison/gochi/pkg/types"
)

func TestNewDataCache(t *testing.T) {
	maxSize := int64(1024 * 1024) // 1MB
	ttl := 5 * time.Minute

	cache := NewDataCache(maxSize, ttl)

	if cache == nil {
		t.Fatal("NewDataCache returned nil")
	}

	if cache.maxSize != maxSize {
		t.Errorf("maxSize = %d, want %d", cache.maxSize, maxSize)
	}

	if cache.defaultTTL != ttl {
		t.Errorf("defaultTTL = %v, want %v", cache.defaultTTL, ttl)
	}
}

func TestCacheSetAndGet(t *testing.T) {
	cache := NewDataCache(1024*1024, 5*time.Minute)

	key := "test_key"
	data := "test_data"
	size := int64(100)

	// Set
	cache.Set(key, data, size)

	// Get
	result, found := cache.Get(key)
	if !found {
		t.Fatal("Get should find the key")
	}

	if result != data {
		t.Errorf("Get returned %v, want %v", result, data)
	}
}

func TestCacheMiss(t *testing.T) {
	cache := NewDataCache(1024*1024, 5*time.Minute)

	_, found := cache.Get("nonexistent_key")
	if found {
		t.Error("Get should not find nonexistent key")
	}

	stats := cache.GetStats()
	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}
}

func TestCacheHitRate(t *testing.T) {
	cache := NewDataCache(1024*1024, 5*time.Minute)

	cache.Set("key1", "data1", 100)

	// 1 hit
	cache.Get("key1")

	// 1 miss
	cache.Get("key2")

	hitRate := cache.GetHitRate()
	expected := 0.5 // 1 hit out of 2 total accesses

	if hitRate != expected {
		t.Errorf("HitRate = %f, want %f", hitRate, expected)
	}
}

func TestCacheEviction(t *testing.T) {
	// Create small cache
	cache := NewDataCache(200, 5*time.Minute) // 200 bytes max

	// Add entries that exceed max size
	cache.Set("key1", "data1", 100)
	cache.Set("key2", "data2", 100)
	cache.Set("key3", "data3", 100) // This should trigger eviction of key1

	// key1 should be evicted (LRU)
	_, found := cache.Get("key1")
	if found {
		t.Error("key1 should have been evicted")
	}

	// key2 and key3 should still exist
	_, found = cache.Get("key2")
	if !found {
		t.Error("key2 should still exist")
	}

	_, found = cache.Get("key3")
	if !found {
		t.Error("key3 should still exist")
	}

	stats := cache.GetStats()
	if stats.Evictions < 1 {
		t.Errorf("Evictions = %d, expected at least 1", stats.Evictions)
	}
}

func TestCacheLRU(t *testing.T) {
	cache := NewDataCache(200, 5*time.Minute)

	cache.Set("key1", "data1", 100)
	cache.Set("key2", "data2", 100)

	// Access key1 to make it recently used
	cache.Get("key1")

	// Add key3, which should evict key2 (least recently used)
	cache.Set("key3", "data3", 100)

	// key2 should be evicted
	_, found := cache.Get("key2")
	if found {
		t.Error("key2 should have been evicted (LRU)")
	}

	// key1 should still exist (recently accessed)
	_, found = cache.Get("key1")
	if !found {
		t.Error("key1 should still exist")
	}
}

func TestCacheTTL(t *testing.T) {
	cache := NewDataCache(1024*1024, 100*time.Millisecond) // 100ms TTL

	cache.Set("key1", "data1", 100)

	// Should exist immediately
	_, found := cache.Get("key1")
	if !found {
		t.Error("key should exist immediately")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found = cache.Get("key1")
	if found {
		t.Error("key should be expired after TTL")
	}
}

func TestCacheSetWithCustomTTL(t *testing.T) {
	cache := NewDataCache(1024*1024, 5*time.Minute)

	cache.SetWithTTL("key1", "data1", 100, 50*time.Millisecond)

	// Should exist immediately
	_, found := cache.Get("key1")
	if !found {
		t.Error("key should exist immediately")
	}

	// Wait for custom TTL
	time.Sleep(70 * time.Millisecond)

	// Should be expired
	_, found = cache.Get("key1")
	if found {
		t.Error("key should be expired after custom TTL")
	}
}

func TestCacheDelete(t *testing.T) {
	cache := NewDataCache(1024*1024, 5*time.Minute)

	cache.Set("key1", "data1", 100)

	// Verify exists
	_, found := cache.Get("key1")
	if !found {
		t.Error("key should exist")
	}

	// Delete
	cache.Delete("key1")

	// Should not exist
	_, found = cache.Get("key1")
	if found {
		t.Error("key should not exist after delete")
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewDataCache(1024*1024, 5*time.Minute)

	cache.Set("key1", "data1", 100)
	cache.Set("key2", "data2", 100)
	cache.Set("key3", "data3", 100)

	// Clear
	cache.Clear()

	// All should be gone
	_, found := cache.Get("key1")
	if found {
		t.Error("cache should be empty after clear")
	}

	stats := cache.GetStats()
	if stats.EntryCount != 0 {
		t.Errorf("EntryCount = %d, want 0 after clear", stats.EntryCount)
	}

	if stats.CurrentSize != 0 {
		t.Errorf("CurrentSize = %d, want 0 after clear", stats.CurrentSize)
	}
}

func TestCacheCleanup(t *testing.T) {
	cache := NewDataCache(1024*1024, 50*time.Millisecond)

	cache.Set("key1", "data1", 100)
	cache.Set("key2", "data2", 100)
	cache.Set("key3", "data3", 100)

	// Wait for expiration
	time.Sleep(70 * time.Millisecond)

	// Cleanup expired entries
	removed := cache.Cleanup()

	if removed != 3 {
		t.Errorf("Cleanup removed %d entries, want 3", removed)
	}

	stats := cache.GetStats()
	if stats.EntryCount != 0 {
		t.Errorf("EntryCount = %d, want 0 after cleanup", stats.EntryCount)
	}
}

func TestCacheStats(t *testing.T) {
	cache := NewDataCache(1024*1024, 5*time.Minute)

	cache.Set("key1", "data1", 100)
	cache.Set("key2", "data2", 200)

	cache.Get("key1") // Hit
	cache.Get("key3") // Miss

	stats := cache.GetStats()

	if stats.Hits != 1 {
		t.Errorf("Hits = %d, want 1", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Misses = %d, want 1", stats.Misses)
	}

	if stats.CurrentSize != 300 {
		t.Errorf("CurrentSize = %d, want 300", stats.CurrentSize)
	}

	if stats.EntryCount != 2 {
		t.Errorf("EntryCount = %d, want 2", stats.EntryCount)
	}
}

func TestPetCache(t *testing.T) {
	petCache := NewPetCache(10, 30) // 10MB, 30 min TTL

	if petCache == nil {
		t.Fatal("NewPetCache returned nil")
	}

	petID := types.PetID("test_pet")
	petData := map[string]interface{}{
		"name": "Fluffy",
		"age":  3,
	}

	// Cache pet
	petCache.CachePet(petID, petData, 1024)

	// Get pet
	result, found := petCache.GetPet(petID)
	if !found {
		t.Error("Pet should be found in cache")
	}

	resultMap := result.(map[string]interface{})
	if resultMap["name"] != "Fluffy" {
		t.Error("Pet data mismatch")
	}

	// Invalidate
	petCache.InvalidatePet(petID)

	// Should not be found
	_, found = petCache.GetPet(petID)
	if found {
		t.Error("Pet should not be found after invalidation")
	}
}

func TestPetCacheCleanup(t *testing.T) {
	petCache := NewPetCache(10, 1) // 1 minute TTL

	// Set custom TTL for testing
	petCache.cache.SetWithTTL("pet1", "data1", 100, 50*time.Millisecond)

	// Wait for expiration
	time.Sleep(70 * time.Millisecond)

	// Cleanup
	removed := petCache.CleanupExpired()

	if removed != 1 {
		t.Errorf("CleanupExpired removed %d entries, want 1", removed)
	}
}

func TestCacheUpdate(t *testing.T) {
	cache := NewDataCache(1024*1024, 5*time.Minute)

	// Set initial value
	cache.Set("key1", "data1", 100)

	// Update with new value and size
	cache.Set("key1", "new_data", 200)

	// Should have updated value
	result, found := cache.Get("key1")
	if !found {
		t.Error("key should exist after update")
	}

	if result != "new_data" {
		t.Errorf("Got %v, want 'new_data'", result)
	}

	// Size should be updated
	stats := cache.GetStats()
	if stats.CurrentSize != 200 {
		t.Errorf("CurrentSize = %d, want 200", stats.CurrentSize)
	}
}
