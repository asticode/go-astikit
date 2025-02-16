package astikit

import (
	"testing"
)

type cacheItem int

func (i cacheItem) Size() int { return int(i) }

func cacheFunc(i int) func(i CacheItem) bool {
	return func(ci CacheItem) bool { return int(ci.(cacheItem)) == i }
}

func TestCache(t *testing.T) {
	// Cache can be disabled
	c := NewCache(CacheOptions{})
	c.Set(cacheItem(1))
	_, ok := c.Get(cacheFunc(1))
	if ok {
		t.Fatal("expected false, got true")
	}

	// Cache can be limited
	c = NewCache(CacheOptions{MaxSize: 5})
	if _, ok = c.Get(cacheFunc(1)); ok {
		t.Fatal("expected false, got true")
	}
	c.Set(cacheItem(1))
	i, ok := c.Get(cacheFunc(1))
	if !ok {
		t.Fatal("expected true, got false")
	}
	if e, g := 1, int(i.(cacheItem)); e != g {
		t.Fatalf("expected %d, got %d", e, g)
	}
	c.Set(cacheItem(2))
	c.Set(cacheItem(3))
	if _, ok = c.Get(cacheFunc(1)); ok {
		t.Fatal("expected false, got true")
	}
	if _, ok = c.Get(cacheFunc(3)); !ok {
		t.Fatal("expected true, got false")
	}
	// Getting an item makes it less likely to get purged
	if _, ok = c.Get(cacheFunc(2)); !ok {
		t.Fatal("expected true, got false")
	}
	c.Set(cacheItem(1))
	if _, ok = c.Get(cacheFunc(3)); ok {
		t.Fatal("expected false, got true")
	}
	if _, ok = c.Get(cacheFunc(1)); !ok {
		t.Fatal("expected true, got false")
	}
	if _, ok = c.Get(cacheFunc(2)); !ok {
		t.Fatal("expected true, got false")
	}
	c.Set(cacheItem(6))
	if _, ok = c.Get(cacheFunc(6)); ok {
		t.Fatal("expected false, got true")
	}

	// Cache can be unlimited
	c = NewCache(CacheOptions{MaxSize: -1})
	c.Set(cacheItem(1))
	c.Set(cacheItem(2))
	c.Set(cacheItem(3))
	if _, ok = c.Get(cacheFunc(1)); !ok {
		t.Fatal("expected true, got false")
	}
	if _, ok = c.Get(cacheFunc(2)); !ok {
		t.Fatal("expected true, got false")
	}
	if _, ok = c.Get(cacheFunc(3)); !ok {
		t.Fatal("expected true, got false")
	}
	c.Delete(cacheFunc(2))
	if _, ok = c.Get(cacheFunc(1)); !ok {
		t.Fatal("expected true, got false")
	}
	if _, ok = c.Get(cacheFunc(2)); ok {
		t.Fatal("expected false, got true")
	}
	if _, ok = c.Get(cacheFunc(3)); !ok {
		t.Fatal("expected true, got false")
	}
}
