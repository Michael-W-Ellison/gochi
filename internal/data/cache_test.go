package data

import (
	"testing"
	"time"
)

// testCache returns a cache on a clock the test moves
func testCache(maxSize int64, ttl time.Duration) (*Cache, *time.Time) {
	c := NewCache(maxSize, ttl)
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return clock }
	return c, &clock
}

func TestCacheGetSet(t *testing.T) {
	c, _ := testCache(100, 0)
	if _, ok := c.Get("a"); ok {
		t.Fatal("An empty cache has nothing")
	}
	c.Set("a", "apple", 10)
	if v, ok := c.Get("a"); !ok || v != "apple" {
		t.Errorf("Get(a) = %v, %v", v, ok)
	}
	c.Set("a", "apricot", 20)
	if v, _ := c.Get("a"); v != "apricot" || c.Stats().Size != 20 || c.Stats().Entries != 1 {
		t.Errorf("Replacing should update the value and size: %v, %+v", v, c.Stats())
	}
	c.Delete("a")
	if _, ok := c.Get("a"); ok || c.Stats().Size != 0 {
		t.Errorf("Deleted entries are gone: %+v", c.Stats())
	}

	stats := c.Stats()
	if stats.Hits != 2 || stats.Misses != 2 || stats.HitRate() != 0.5 {
		t.Errorf("Stats = %+v, hit rate %.2f", stats, stats.HitRate())
	}
}

func TestCacheEvictsLeastRecentlyUsed(t *testing.T) {
	c, _ := testCache(30, 0)
	c.Set("a", 1, 10)
	c.Set("b", 2, 10)
	c.Set("c", 3, 10)
	c.Get("a") // b is now the least recently used
	c.Set("d", 4, 10)

	if _, ok := c.Get("b"); ok {
		t.Error("b should have been evicted")
	}
	for _, key := range []string{"a", "c", "d"} {
		if _, ok := c.Get(key); !ok {
			t.Errorf("%s should still be cached", key)
		}
	}
	if s := c.Stats(); s.Evictions != 1 || s.Size != 30 {
		t.Errorf("Stats = %+v", s)
	}

	c.Set("huge", 5, 31)
	if _, ok := c.Get("huge"); ok || c.Stats().Entries != 3 {
		t.Error("A value larger than the cache should not be stored or evict others")
	}
}

func TestCacheExpiry(t *testing.T) {
	c, clock := testCache(100, time.Minute)
	c.Set("a", 1, 10)
	c.SetWithTTL("b", 2, 10, time.Hour)
	c.SetWithTTL("c", 3, 10, 0) // Never expires

	*clock = clock.Add(time.Minute)
	if _, ok := c.Get("a"); ok {
		t.Error("a should have expired")
	}
	if removed := c.Cleanup(); removed != 0 {
		t.Errorf("Nothing else has expired yet, removed %d", removed)
	}
	*clock = clock.Add(time.Hour)
	if removed := c.Cleanup(); removed != 1 || c.Stats().Entries != 1 {
		t.Errorf("Cleanup should remove b only: removed %d, %+v", removed, c.Stats())
	}
	if _, ok := c.Get("c"); !ok {
		t.Error("c never expires")
	}

	c.Clear()
	if s := c.Stats(); s.Entries != 0 || s.Size != 0 {
		t.Errorf("Clear should empty the cache: %+v", s)
	}
}
