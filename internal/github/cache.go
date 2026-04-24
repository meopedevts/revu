package github

import (
	"strings"
	"sync"
	"time"
)

// memCache is a small TTL-keyed cache for GetPRFullDetails and GetPRDiff
// results. It absorbs repeat clicks on the same PR in the details view
// without hammering `gh`. Keys are "namespace:url" strings; invalidate(url)
// clears every namespace for a single PR after a successful merge.
type memCache struct {
	mu  sync.Mutex
	ttl time.Duration
	now func() time.Time
	m   map[string]cacheEntry
}

type cacheEntry struct {
	value  any
	expiry time.Time
}

func newMemCache(ttl time.Duration) *memCache {
	return &memCache{
		ttl: ttl,
		now: time.Now,
		m:   make(map[string]cacheEntry),
	}
}

func (c *memCache) get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok {
		return nil, false
	}
	if c.now().After(e.expiry) {
		delete(c.m, key)
		return nil, false
	}
	return e.value, true
}

func (c *memCache) set(key string, v any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = cacheEntry{value: v, expiry: c.now().Add(c.ttl)}
}

// invalidate drops every entry whose key contains url. Used after a merge so
// the next fetch reflects the new state instead of a 30-second-old snapshot.
func (c *memCache) invalidate(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.m {
		if strings.Contains(k, url) {
			delete(c.m, k)
		}
	}
}
