package bluray

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const cacheTTL = 2 * time.Hour

type cache struct {
	mu        sync.Mutex
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
	c.mu.Lock()
	defer c.mu.Unlock()

	if page == 0 && !c.createdAt.IsZero() && time.Now().After(c.createdAt.Add(cacheTTL)) {
		c.flush()
		return nil, false
	}

	releases, ok := c.pages[page]
	return releases, ok
}

// setPage stores releases for the given page. Each release with a cover image
// URL gets a random hex ID assigned; the URL mapping is recorded so the image
// proxy can resolve it later. The raw upstream URL is cleared from the slice
// elements in-place, so callers must not use the original ImageURL values after
// this call. For page 0, the cache creation timestamp is recorded here.
func (c *cache) setPage(page int, releases []Release) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range releases {
		if releases[i].ImageURL != "" {
			id := newHexID()
			c.images[id] = releases[i].ImageURL
			releases[i].ImageID = id
			releases[i].ImageURL = ""
		}
	}

	c.pages[page] = releases

	if page == 0 {
		c.createdAt = time.Now()
	}
}

// resolveImage returns the upstream cover URL for the given hex ID.
func (c *cache) resolveImage(id string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
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
