package bluray

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const detailConcurrency = 3

// Service fetches and returns Blu-ray release data.
type Service struct {
	scraper *scraper
}

// New creates a new Service.
func New() *Service {
	return &Service{scraper: newScraper()}
}

// Releases returns all releases from the given listing page, fetching detail
// pages with limited concurrency. Only releases from the current year or the
// previous year are included.
func (s *Service) Releases(ctx context.Context, page int) ([]Release, error) {
	entries, err := s.scraper.fetchListingPage(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("fetch listing page %d: %w", page, err)
	}

	currentYear := time.Now().Year()

	// Pre-filter: skip entries whose release date is outside the allowed window
	// before firing off expensive detail-page requests.
	var filtered []listingEntry
	for _, e := range entries {
		year := parseYearFromDate(e.releaseDate)
		if year >= currentYear-1 {
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

	// Collect successful results in original order.
	var result []Release
	for i, r := range releases {
		if errs[i] != nil {
			continue
		}
		result = append(result, r)
	}

	return result, nil
}
