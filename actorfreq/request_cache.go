package actorfreq

import (
	"container/list"
	"log/slog"
	"sync"
	"time"
	"unsafe"
)

type cacheItem struct {
	key        string
	value      []actorDetails
	expiration int64
	size       int

	element *list.Element
}

type Cache struct {
	items     map[string]*cacheItem
	order     *list.List
	mutex     sync.RWMutex
	totalSize int
	maxSize   int
}

var requestCache = &Cache{
	items:   make(map[string]*cacheItem),
	order:   list.New(),
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

func (c *Cache) remove(item *cacheItem) {
	c.totalSize -= item.size
	c.order.Remove(item.element)
	delete(c.items, item.key)
}

func (c *Cache) set(key string, value []actorDetails, duration time.Duration) {
	size := calculateSize(value)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if size < c.maxSize {
		// Remove existing item if key already exists
		if item, found := c.items[key]; found {
			c.remove(item)
		}

		// Ensure cache does not exceed max size
		for c.totalSize+size > c.maxSize {
			if oldest := c.order.Front(); oldest != nil {
				key := oldest.Value.(string)
				if item, found := c.items[key]; found {
					c.remove(item)
				}
			}
		}

		expiration := time.Now().Add(duration).UnixNano()
		element := c.order.PushBack(key)
		c.items[key] = &cacheItem{key: key, value: value, expiration: expiration, size: size, element: element}
		c.totalSize += size
		slog.Info("Request cache updated", "numItems", len(c.items), "totalSize", c.totalSize)
	} else {
		slog.Info("Value too large for request cache", "key", key, "size", size, "maxSize", c.maxSize)
	}
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
	for _, item := range c.items {
		if time.Now().UnixNano() > item.expiration {
			c.remove(item)
		}
	}
	c.mutex.Unlock()
}

func (c *Cache) evictAll() {
	c.mutex.Lock()
	for _, item := range c.items {
		c.remove(item)
	}
	c.mutex.Unlock()
}
