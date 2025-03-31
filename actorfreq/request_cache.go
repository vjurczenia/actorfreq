package actorfreq

import (
	"sync"
	"time"
	"unsafe"
)

type cacheItem struct {
	value      []actorDetails
	expiration int64
	size       int
}

type Cache struct {
	items     map[string]cacheItem
	mutex     sync.RWMutex
	totalSize int
	maxSize   int
}

var requestCache = &Cache{
	items:   make(map[string]cacheItem),
	maxSize: 100 * 1024 * 1024, // 100MB
}

func calculateSize(actors []actorDetails) int {
	totalSize := int(0)

	for _, actor := range actors {
		// Size of the struct itself
		totalSize += int(unsafe.Sizeof(actor))

		// Size of the string fields in actorDetails
		totalSize += len(actor.Name)

		// Size of the Movies slice metadata (slice header)
		totalSize += int(unsafe.Sizeof(actor.Movies))

		for _, movie := range actor.Movies {
			// Size of movieDetails struct
			totalSize += int(unsafe.Sizeof(movie))

			// Size of the string fields in movieDetails
			totalSize += len(movie.FilmSlug)
			totalSize += len(movie.Title)
			totalSize += len(movie.Roles)
		}
	}

	return totalSize
}

func (c *Cache) set(key string, value []actorDetails, duration time.Duration) {
	size := calculateSize(value)
	expiration := time.Now().Add(duration).UnixNano()
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Remove existing item size if key already exists
	if item, found := c.items[key]; found {
		c.totalSize -= item.size
	}

	// Ensure cache does not exceed max size
	for c.totalSize+size > c.maxSize {
		c.evict()
	}

	c.items[key] = cacheItem{value: value, expiration: expiration, size: size}
	c.totalSize += size
}

func (c *Cache) get(key string) ([]actorDetails, bool) {
	c.mutex.RLock()
	item, found := c.items[key]
	c.mutex.RUnlock()
	if !found || time.Now().UnixNano() > item.expiration {
		return nil, false
	}
	return item.value, true
}

func (c *Cache) evict() {
	c.mutex.Lock()
	for key, item := range c.items {
		if time.Now().UnixNano() > item.expiration || c.totalSize > c.maxSize {
			c.totalSize -= item.size
			delete(c.items, key)
		}
	}
	c.mutex.Unlock()
}
