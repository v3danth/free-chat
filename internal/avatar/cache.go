package avatar

import (
	"slices"
	"sync"
	"sync/atomic"

	"github.com/v3danth/free-chat/internal/httpx"
)

// A face depends only on its name and how it is drawn, and drawing one
// costs 0.3 ms as SVG and 1.4 ms as PNG. A room of 200 people asks for 200
// faces per visitor, so drawn faces are kept under a byte budget, which
// also caps what requests for arbitrary names can cost.
const renderBudget = 32 << 20

// Eviction is Redis's approximated LRU: a hit only stamps the entry with a
// logical clock, so concurrent hits share a read lock instead of queueing
// to reorder a list. To evict, sample a few entries, keep the idlest in a
// small pool across rounds, and drop the idlest of the pool. Redis finds
// five samples and a pool of sixteen close to true LRU.
const (
	evictionSamples = 5
	evictionPool    = 16
)

type renderCache struct {
	clock  atomic.Uint64
	mu     sync.RWMutex
	budget int
	used   int
	byKey  map[string]*rendered
	pool   []*rendered // eviction candidates, at most evictionPool
}

type rendered struct {
	key      string
	lastUsed atomic.Uint64
	httpx.Payload
}

func newRenderCache(budget int) *renderCache {
	return &renderCache{budget: budget, byKey: map[string]*rendered{}}
}

// payload returns the drawn face for key, drawing it with draw on a miss.
// Two requests missing at once may both draw; the result is the same.
func (c *renderCache) payload(key string, draw func() (httpx.Payload, error)) (httpx.Payload, error) {
	c.mu.RLock()
	r, ok := c.byKey[key]
	if ok {
		r.lastUsed.Store(c.clock.Add(1))
	}
	c.mu.RUnlock()
	if ok {
		return r.Payload, nil
	}

	p, err := draw()
	if err != nil || p.Size() > c.budget {
		return p, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.byKey[key]; !ok {
		r := &rendered{key: key, Payload: p}
		r.lastUsed.Store(c.clock.Add(1))
		c.byKey[key] = r
		c.used += p.Size()
	}
	for c.used > c.budget {
		c.evictLocked()
	}
	return p, nil
}

// evictLocked drops one entry, the idlest sampling can find. Go randomises
// where map iteration starts, which makes the first few entries a cheap
// sample. The victim is judged by its stamp now, not when it joined the
// pool, so an entry hit since then is spared.
func (c *renderCache) evictLocked() {
	n := 0
	for _, r := range c.byKey {
		if n == evictionSamples {
			break
		}
		c.consider(r)
		n++
	}
	idlest := 0
	for i, r := range c.pool {
		if r.lastUsed.Load() < c.pool[idlest].lastUsed.Load() {
			idlest = i
		}
	}
	victim := c.pool[idlest]
	c.pool = slices.Delete(c.pool, idlest, idlest+1)
	delete(c.byKey, victim.key)
	c.used -= victim.Size()
}

// consider adds r to the eviction pool. A full pool trades its most
// recently used member for r when r is idler.
func (c *renderCache) consider(r *rendered) {
	if slices.Contains(c.pool, r) {
		return
	}
	if len(c.pool) < evictionPool {
		c.pool = append(c.pool, r)
		return
	}
	freshest := 0
	for i, p := range c.pool {
		if p.lastUsed.Load() > c.pool[freshest].lastUsed.Load() {
			freshest = i
		}
	}
	if r.lastUsed.Load() < c.pool[freshest].lastUsed.Load() {
		c.pool[freshest] = r
	}
}
