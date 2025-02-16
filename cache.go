package astikit

import (
	"sync"
)

// Cache is an object capable of caching stuff while ensuring cumulated cached size never gets above
// a provided threshold
type Cache struct {
	items []CacheItem // We use a slice since we want to reorder items when one has been used
	m     sync.Mutex  // Locks items
	o     CacheOptions
	size  int
}

type CacheItem interface {
	Size() int
}

type CacheOptions struct {
	// - 0 disables cache
	// - < 0 disables max size
	MaxSize int
}

func NewCache(o CacheOptions) *Cache {
	return &Cache{o: o}
}

func (c *Cache) Get(found func(i CacheItem) bool) (CacheItem, bool) {
	// Lock
	c.m.Lock()
	defer c.m.Unlock()

	// Find item
	var idx int
	for idx = 0; idx < len(c.items); idx++ {
		if found(c.items[idx]) {
			break
		}
	}

	// Item was not found
	if idx >= len(c.items) {
		return nil, false
	}

	// Save item
	i := c.items[idx]

	// Move entry to the last position
	c.items = append(append(c.items[:idx], c.items[idx+1:]...), i)
	return i, true
}

func (c *Cache) Set(i CacheItem) {
	// Nothing to do
	if c.o.MaxSize == 0 {
		return
	}

	// Item is bigger than cache max size
	if c.o.MaxSize > 0 && i.Size() > c.o.MaxSize {
		return
	}

	// Lock
	c.m.Lock()
	defer c.m.Unlock()

	// Make room for item
	if c.o.MaxSize > 0 {
		for c.size+i.Size() > c.o.MaxSize {
			c.size -= c.items[0].Size()
			c.items = c.items[1:]
		}
	}

	// Store image
	c.size += i.Size()
	c.items = append(c.items, i)
}

func (c *Cache) Delete(remove func(i CacheItem) bool) {
	// Lock
	c.m.Lock()
	defer c.m.Unlock()

	// Loop through entries
	for idx := 0; idx < len(c.items); idx++ {
		// Remove
		if remove(c.items[idx]) {
			c.items = append(c.items[:idx], c.items[idx+1:]...)
			idx--
		}
	}
}
