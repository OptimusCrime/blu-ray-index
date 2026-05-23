package bluray

import (
	"context"
	"fmt"
	"testing"
)

func TestLiveFetch(t *testing.T) {
	s := newScraper()
	releases, err := s.fetchListingPage(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	zeroYear := 0
	for _, r := range releases {
		if r.ProductionYear == 0 {
			zeroYear++
		}
	}
	fmt.Printf("Total: %d, zero productionYear: %d\n", len(releases), zeroYear)
	for i, r := range releases {
		if i >= 8 { break }
		fmt.Printf("  title=%q studio=%q year=%d releaseDate=%q runtime=%q rating=%q genres=%v\n",
			r.Title, r.Studio, r.ProductionYear, r.ReleaseDate, r.Runtime, r.Rating, r.Genres)
	}
}
