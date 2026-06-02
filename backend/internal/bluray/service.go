package bluray

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

const (
	maxPage        = 30
	pagesPerAPIPage = 5
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

// Page 0 acts as the cache gatekeeper: requesting it after the TTL expires
// flushes everything and triggers a fresh scrape.
//
// Each API page maps to pagesPerAPIPage consecutive upstream pages fetched in
// parallel: API page 0 → upstream pages 0–4, page 1 → 5–9, etc.
func (s *Service) Releases(ctx context.Context, page int) ([]Release, error) {
	if cached, ok := s.cache.getPage(page); ok {
		return cached, nil
	}

	startPage := page * pagesPerAPIPage

	type pageResult struct {
		releases []Release
		err      error
	}

	results := make([]pageResult, pagesPerAPIPage)
	var wg sync.WaitGroup
	wg.Add(pagesPerAPIPage)

	for i := 0; i < pagesPerAPIPage; i++ {
		i := i
		go func() {
			defer wg.Done()
			releases, err := s.scraper.fetchListingPage(ctx, startPage+i)
			results[i] = pageResult{releases: releases, err: err}
		}()
	}

	wg.Wait()

	for _, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("fetch listing page: %w", r.err)
		}
	}

	seen := map[string]bool{}
	var merged []Release
	for _, r := range results {
		for _, rel := range r.releases {
			if !seen[rel.ProductID] {
				seen[rel.ProductID] = true
				merged = append(merged, rel)
			}
		}
	}

	currentYear := time.Now().Year()
	var filtered []Release
	for _, r := range merged {
		if r.ProductionYear == 0 || r.ProductionYear >= currentYear-1 {
			filtered = append(filtered, r)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return parseSortDate(filtered[i].ReleaseDate).After(parseSortDate(filtered[j].ReleaseDate))
	})

	return s.cache.setPage(page, filtered), nil
}

func parseSortDate(s string) time.Time {
	for _, layout := range []string{"Jan 2, 2006", "January 2, 2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (s *Service) ResolveImage(id string) (string, bool) {
	return s.cache.resolveImage(id)
}
