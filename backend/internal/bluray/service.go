package bluray

import (
	"context"
	"fmt"
	"time"
)

const maxPage = 150

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

// Page 0 acts as the cache gatekeeper: requesting it after the TTL expires
// flushes everything and triggers a fresh scrape.
func (s *Service) Releases(ctx context.Context, page int) ([]Release, error) {
	if cached, ok := s.cache.getPage(page); ok {
		return cached, nil
	}

	releases, err := s.scraper.fetchListingPage(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("fetch listing page %d: %w", page, err)
	}

	currentYear := time.Now().Year()
	isRecentEnough := func(year int) bool {
		return year == 0 || year >= currentYear-1
	}

	var result []Release
	for _, r := range releases {
		if isRecentEnough(r.ProductionYear) {
			result = append(result, r)
		}
	}

	return s.cache.setPage(page, result), nil
}

func (s *Service) ResolveImage(id string) (string, bool) {
	return s.cache.resolveImage(id)
}
