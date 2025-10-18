package selfcach

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type item struct {
	value []byte
	ttl   int64
}

type Cache struct {
	logger *zerolog.Logger

	cacheMap   map[string]item
	mx         sync.Mutex
	quit       chan struct{}
	defaultTTL time.Duration
}

func NewCache(logger *zerolog.Logger, defaultTTL, cleanupInterval time.Duration) *Cache {
	c := &Cache{
		cacheMap:   make(map[string]item),
		quit:       make(chan struct{}),
		defaultTTL: defaultTTL,
		logger:     logger,
	}

	go func() {
		ticker := time.NewTicker(cleanupInterval)
		for {
			select {
			case <-ticker.C:
				c.cleanup()
			case <-c.quit:
				ticker.Stop()
				return
			}
		}
	}()

	return c
}

func (c *Cache) Set(key string, data []byte) error {
	c.mx.Lock()
	defer c.mx.Unlock()

	if key == "" || data == nil {
		return fmt.Errorf("cache key/value is invalid")
	}

	var expiry int64
	if c.defaultTTL > 0 {
		expiry = time.Now().Add(c.defaultTTL).UnixNano()
	}

	c.cacheMap[key] = item{value: data, ttl: expiry}
	return nil
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mx.Lock()
	defer c.mx.Unlock()

	it, exists := c.cacheMap[key]
	if !exists {
		return nil, false
	}

	if it.ttl > 0 && time.Now().UnixNano() > it.ttl {
		delete(c.cacheMap, key)
		return nil, false
	}

	return it.value, true
}

func (c *Cache) Del(key string) bool {
	c.mx.Lock()
	defer c.mx.Unlock()

	_, exists := c.cacheMap[key]
	if !exists {
		return false
	}
	delete(c.cacheMap, key)
	return true
}

func (c *Cache) cleanup() {
	c.mx.Lock()
	defer c.mx.Unlock()

	now := time.Now().UnixNano()
	for k, it := range c.cacheMap {
		if it.ttl > 0 && now > it.ttl {
			delete(c.cacheMap, k)
		}
	}
}

func (c *Cache) StopCleanup() {
	close(c.quit)
}
