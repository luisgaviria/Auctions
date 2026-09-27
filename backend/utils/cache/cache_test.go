package cache

import (
	"testing"
	"time"
)

func TestCacheStoresAndExpiresValues(t *testing.T) {
	key := "test_cache_value"
	t.Cleanup(func() { Cache.Delete(key) })

	Cache.Set(key, "value", time.Minute)
	got, found := Cache.Get(key)
	if !found || got != "value" {
		t.Fatalf("Cache.Get() = (%v, %t), want (value, true)", got, found)
	}

	Cache.Delete(key)
	if _, found := Cache.Get(key); found {
		t.Error("deleted cache key was still present")
	}
}
