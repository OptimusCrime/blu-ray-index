package bluray

import (
	"testing"
	"time"
)

func TestCacheSetPageAndResolveImage(t *testing.T) {
	c := newCache()

	releases := []Release{
		{ProductID: "1", Title: "Movie A", ImageURL: "https://images.static-bluray.com/a.jpg"},
		{ProductID: "2", Title: "Movie B", ImageURL: ""},
	}

	c.setPage(0, releases)

	// ImageURL should be cleared and ImageID assigned for the release that had a URL.
	if releases[0].ImageURL != "" {
		t.Error("ImageURL should be cleared after setPage")
	}
	if releases[0].ImageID == "" {
		t.Error("ImageID should be set after setPage")
	}
	// Release without image should remain unchanged.
	if releases[1].ImageID != "" {
		t.Error("ImageID should not be set for release with no image")
	}

	// resolveImage should return the original URL.
	url, ok := c.resolveImage(releases[0].ImageID)
	if !ok {
		t.Fatal("resolveImage returned not found")
	}
	if url != "https://images.static-bluray.com/a.jpg" {
		t.Errorf("resolveImage = %q, want original URL", url)
	}
}

func TestCacheHitAndMiss(t *testing.T) {
	c := newCache()

	_, ok := c.getPage(1)
	if ok {
		t.Fatal("expected cache miss for page 1 before any set")
	}

	c.setPage(1, []Release{{Title: "Cached"}})

	cached, ok := c.getPage(1)
	if !ok {
		t.Fatal("expected cache hit for page 1 after set")
	}
	if len(cached) != 1 || cached[0].Title != "Cached" {
		t.Errorf("unexpected cached value: %+v", cached)
	}
}

func TestCacheTTLFlushOnPageZero(t *testing.T) {
	c := newCache()

	c.setPage(0, []Release{{Title: "Old Release"}})
	c.setPage(1, []Release{{Title: "Page 1 Release"}})

	// Backdate createdAt so TTL is expired.
	c.mu.Lock()
	c.createdAt = time.Now().Add(-(cacheTTL + time.Second))
	c.mu.Unlock()

	// Requesting page 0 after TTL should flush and return a miss.
	_, ok := c.getPage(0)
	if ok {
		t.Fatal("expected cache miss for page 0 after TTL expiry")
	}

	// Page 1 should also be gone after flush.
	_, ok = c.getPage(1)
	if ok {
		t.Fatal("expected page 1 to be flushed after TTL expiry")
	}
}

func TestCacheTTLNotFlushBeforeExpiry(t *testing.T) {
	c := newCache()
	c.setPage(0, []Release{{Title: "Fresh Release"}})

	// TTL not expired — page 0 should still be a hit.
	cached, ok := c.getPage(0)
	if !ok {
		t.Fatal("expected cache hit for page 0 within TTL")
	}
	if len(cached) == 0 {
		t.Error("expected non-empty cached page")
	}
}
