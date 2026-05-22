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

// Service fetches and returns Blu-ray release data.
type Service struct {
	scraper *scraper
	cache   *cache
}

// New creates a new Service.
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

	var filtered []listingEntry
	for _, e := range entries {
		if e.productionYear == 0 || e.productionYear >= currentYear-1 {
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
			sem <- struct{}{}
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
		if r.ProductionYear != 0 && r.ProductionYear < currentYear-1 {
			continue
		}
		result = append(result, r)
	}

	s.cache.setPage(page, result)
	return result, nil
}

// ResolveImage returns the upstream cover image URL for the given hex ID.
func (s *Service) ResolveImage(id string) (string, bool) {
	return s.cache.resolveImage(id)
}
