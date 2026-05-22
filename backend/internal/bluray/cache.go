package bluray

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const cacheTTL = 2 * time.Hour

type cache struct {
	mu        sync.RWMutex
	createdAt time.Time
	pages     map[int][]Release
	images    map[string]string // hex id → upstream cover URL
}

func newCache() *cache {
	return &cache{
		pages:  make(map[int][]Release),
		images: make(map[string]string),
	}
}

// getPage returns cached releases for the given page.
// When page 0 is requested and the TTL has elapsed, the entire cache is flushed
// and a miss is returned so the caller re-scrapes from scratch.
func (c *cache) getPage(page int) ([]Release, bool) {
	// Page 0 may trigger a cache flush, which requires a write lock.
	if page == 0 {
		c.mu.Lock()
		defer c.mu.Unlock()
		if !c.createdAt.IsZero() && time.Now().After(c.createdAt.Add(cacheTTL)) {
			c.flush()
			return nil, false
		}
		releases, ok := c.pages[page]
		return releases, ok
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	releases, ok := c.pages[page]
	return releases, ok
}

func (c *cache) setPage(page int, releases []Release) []Release {
	c.mu.Lock()
	defer c.mu.Unlock()

	stored := make([]Release, len(releases))
	for i, r := range releases {
		if r.ImageURL != "" {
			id := newHexID()
			c.images[id] = r.ImageURL
			r.ImageID = id
			r.ImageURL = ""
		}
		stored[i] = r
	}

	c.pages[page] = stored

	if page == 0 {
		c.createdAt = time.Now()
	}

	return stored
}

func (c *cache) resolveImage(id string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	url, ok := c.images[id]
	return url, ok
}

// flush resets all cache state. Must be called with c.mu held.
func (c *cache) flush() {
	c.createdAt = time.Time{}
	c.pages = make(map[int][]Release)
	c.images = make(map[string]string)
}

func newHexID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
