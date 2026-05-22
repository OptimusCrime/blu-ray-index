package bluray

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const (
	detailConcurrency = 3
	maxPage           = 20
)

type Service struct {
	scraper *scraper
	cache   *cache
}

func New() *Service {
	return &Service{
		scraper: newScraper(),
		cache:   newCache(),
	}
}

// Releases returns all releases from the given listing page. Results are
// served from the in-memory cache when available. Page 0 acts as the cache
// gatekeeper: requesting it after the TTL expires flushes everything and
// triggers a fresh scrape.
func (s *Service) Releases(ctx context.Context, page int) ([]Release, error) {
	if cached, ok := s.cache.getPage(page); ok {
		return cached, nil
	}

	entries, err := s.scraper.fetchListingPage(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("fetch listing page %d: %w", page, err)
	}

	currentYear := time.Now().Year()

	// isRecentEnough returns true when year is unknown (0) or within the
	// allowed window. Applied twice: once to skip listing entries whose
	// listing-page year is clearly old, and again after the detail fetch
	// because the detail page may refine the year (e.g. from a year range).
	isRecentEnough := func(year int) bool {
		return year == 0 || year >= currentYear-1
	}

	var filtered []listingEntry
	for _, e := range entries {
		if isRecentEnough(e.productionYear) {
			filtered = append(filtered, e)
		}
	}

	releases := make([]Release, len(filtered))
	errs := make([]error, len(filtered))

	sem := make(chan struct{}, detailConcurrency)
	var wg sync.WaitGroup

	for i, entry := range filtered {
		wg.Add(1)
		go func(idx int, e listingEntry) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				errs[idx] = ctx.Err()
				return
			}
			defer func() { <-sem }()

			r, err := s.scraper.fetchDetailPage(ctx, e)
			if err != nil {
				slog.Warn("failed to fetch detail page", "url", e.url, "err", err)
				errs[idx] = err
				return
			}
			releases[idx] = *r
		}(i, entry)
	}

	wg.Wait()

	var result []Release
	for i, r := range releases {
		if errs[i] != nil {
			continue
		}
		if !isRecentEnough(r.ProductionYear) {
			continue
		}
		result = append(result, r)
	}

	s.cache.setPage(page, result)
	return result, nil
}

func (s *Service) ResolveImage(id string) (string, bool) {
	return s.cache.resolveImage(id)
}
