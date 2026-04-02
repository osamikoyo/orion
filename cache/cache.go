package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/osamikoyo/orion/logger"
	"go.uber.org/zap"
)

type item struct {
	value []byte
	ttl   int64
}

type cache struct {
	cacheMap   map[string]item
	mx         sync.Mutex
	quit       chan struct{}
	defaultTTL time.Duration
	logger     *logger.Logger
}

func newCache(logger *logger.Logger, defaultTTL, cleanupInterval time.Duration) *cache {
	c := &cache{
		cacheMap:   make(map[string]item),
		quit:       make(chan struct{}),
		defaultTTL: defaultTTL,
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

func (c *cache) Set(key string, data []byte) error {
	c.logger.Info("setting new value in cache",
		zap.String("key", key),
		zap.ByteString("data", data))

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

func (c *cache) Get(key string) ([]byte, bool) {
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

func (c *cache) Del(key string) bool {
	c.mx.Lock()
	defer c.mx.Unlock()

	_, exists := c.cacheMap[key]
	if !exists {
		return false
	}
	delete(c.cacheMap, key)
	return true
}

func (c *cache) cleanup() {
	c.mx.Lock()
	defer c.mx.Unlock()

	now := time.Now().UnixNano()
	for k, it := range c.cacheMap {
		if it.ttl > 0 && now > it.ttl {
			delete(c.cacheMap, k)
		}
	}
}

func (c *cache) StopCleanup() {
	close(c.quit)
}
