package bluray

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/RomainMichau/CycleTLS/cycletls"
	"github.com/RomainMichau/cloudscraper_go/cloudscraper"
	"golang.org/x/net/html/charset"
)

const (
	baseListingURL = "https://www.blu-ray.com/movies/movies.php?show=newreleases&sortby=releasetimestamp"
	userAgent      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
	listingCookie  = "listlayout_7=full"
	requestTimeout = 30

	// blu-ray.com's bot detection intermittently blocks a request based on
	// its TLS fingerprint: sometimes a fully empty response, sometimes a
	// small "error42" body, both with a 200 status. cloudscraper picks a
	// random browser/JA3 profile per Init, so retrying with a fresh client
	// gives later attempts a real chance of getting an accepted fingerprint.
	maxFetchAttempts = 8
	retryBaseDelay   = 500 * time.Millisecond
)

var (
	productIDFromURLRe = regexp.MustCompile(`/(\d+)/$`)
	pipeYearRe         = regexp.MustCompile(`^(\d{4})(?:-(\d{4}))?$`)
	pipeRuntimeRe      = regexp.MustCompile(`^\d+ min$`)
	pipeRatingRe       = regexp.MustCompile(`(?i)^(?:not rated|unrated|rated\s+\S+)$`)
	pipeDateRe         = regexp.MustCompile(`^(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d`)
	titleSuffixRe      = regexp.MustCompile(`(?i)\s+Blu-ray\s*$`)
	yearFallbackRe     = regexp.MustCompile(`\b(\d{4})\b`)
)

type scraper struct{}

func newScraper() *scraper {
	return &scraper{}
}

func (s *scraper) get(ctx context.Context, url string) (*goquery.Document, error) {
	var lastErr error
	for attempt := 1; attempt <= maxFetchAttempts; attempt++ {
		doc, err := s.getOnce(url)
		if err == nil {
			return doc, nil
		}
		lastErr = err
		slog.Warn("bluray: fetch attempt failed", "url", url, "attempt", attempt, "err", err)

		if attempt == maxFetchAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryBaseDelay * time.Duration(attempt)):
		}
	}
	return nil, fmt.Errorf("bluray: GET %s: giving up after %d attempts: %w", url, maxFetchAttempts, lastErr)
}

func (s *scraper) getOnce(url string) (*goquery.Document, error) {
	// A fresh client picks a new random browser/JA3 profile, so each retry
	// gets an independent chance of passing the upstream bot detection.
	client, err := cloudscraper.Init(false, false)
	if err != nil {
		return nil, fmt.Errorf("init client: %w", err)
	}

	options := cycletls.Options{
		Headers: map[string]string{
			"accept-language": "en-US,en;q=0.9",
			"cookie":          listingCookie,
		},
		Timeout: requestTimeout,
	}

	resp, err := client.Do(url, options, "GET")
	if err != nil {
		return nil, fmt.Errorf("GET: %w", err)
	}

	if resp.Status != 200 {
		return nil, fmt.Errorf("status %d", resp.Status)
	}

	if !looksLikeListingPage(resp.Body) {
		return nil, fmt.Errorf("blocked response (%d bytes)", len(resp.Body))
	}

	contentType := resp.Headers["Content-Type"]
	utf8Reader, err := charset.NewReader(strings.NewReader(resp.Body), contentType)
	if err != nil {
		return nil, fmt.Errorf("charset: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(utf8Reader)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return doc, nil
}

// looksLikeListingPage guards against blu-ray.com's bot detection, which
// returns a 200 status with either an empty body or a short "error42" body
// instead of a real page when a request is blocked.
func looksLikeListingPage(body string) bool {
	return strings.Contains(strings.ToLower(body), "<html")
}

// fetchListingPage scrapes all release entries from the given page of the new
// releases listing. The listing is requested in "full" layout (cookie
// listlayout_7=full) so that each entry includes the pipe-separated metadata
// line, a synopsis excerpt, and genre tags inline — no detail-page requests
// are needed.
func (s *scraper) fetchListingPage(ctx context.Context, page int) ([]Release, error) {
	url := baseListingURL
	if page > 0 {
		url = fmt.Sprintf("%s&page=%d", baseListingURL, page)
	}

	doc, err := s.get(ctx, url)
	if err != nil {
		return nil, err
	}

	var releases []Release
	seen := map[string]bool{}

	// Release entries are tables with a td[width='85%'] content column.
	// The outer page layout table also matches this selector via its nested
	// release tables, so we deduplicate by href.
	doc.Find("table").Each(func(_ int, table *goquery.Selection) {
		td := table.Find("td[width='85%']").First()
		if td.Length() == 0 {
			return
		}

		link := td.Find("a.noline").First()
		href, ok := link.Attr("href")
		if !ok || href == "" {
			return
		}

		if seen[href] {
			return
		}
		seen[href] = true

		m := productIDFromURLRe.FindStringSubmatch(href)
		if len(m) < 2 {
			return
		}
		productID := m[1]

		title := cleanTitle(strings.TrimSpace(link.Find("h3").Text()))

		// Cover image: prefer the actual src from the listing img (eager-loaded
		// entries have a real _small.jpg URL); replace the suffix with _large.jpg
		// for higher resolution. Lazy-loaded entries use a transparent placeholder,
		// so fall back to constructing from the product ID.
		imageURL := fmt.Sprintf("https://images.static-bluray.com/movies/covers/%s_large.jpg", productID)
		if img := table.Find("img.cover").First(); img.Length() > 0 {
			if src, _ := img.Attr("src"); strings.HasSuffix(src, "_small.jpg") {
				imageURL = strings.TrimSuffix(src, "_small.jpg") + "_large.jpg"
			}
		}

		// Pipe-separated metadata: "Studio | Year | [Runtime] | [Rating] | [Format] | Date"
		smallText := strings.TrimSpace(td.Find("small").First().Text())
		studio, releaseDate, runtime, rating, productionYear := parsePipeInfo(smallText)
		if releaseDate == "" {
			return
		}

		description := extractListingDescription(td)

		genres := []string{}
		td.Find("a[href*='genre=']").Each(func(_ int, a *goquery.Selection) {
			if g := strings.TrimSpace(a.Text()); g != "" {
				genres = append(genres, g)
			}
		})

		releases = append(releases, Release{
			ProductID:      productID,
			URL:            href,
			Title:          title,
			ReleaseDate:    releaseDate,
			ReleaseYear:    parseYearFromDate(releaseDate),
			ProductionYear: productionYear,
			Studio:         studio,
			Runtime:        runtime,
			Rating:         rating,
			Description:    description,
			Genres:         genres,
			ImageURL:       imageURL,
		})
	})

	return releases, nil
}

func parsePipeInfo(raw string) (studio, releaseDate, runtime, rating string, productionYear int) {
	parts := strings.Split(raw, " | ")
	if len(parts) == 0 {
		return
	}
	studio = strings.TrimSpace(parts[0])

	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		switch {
		case pipeYearRe.MatchString(p):
			m := pipeYearRe.FindStringSubmatch(p)
			if m[2] != "" {
				productionYear, _ = strconv.Atoi(m[2])
			} else {
				productionYear, _ = strconv.Atoi(m[1])
			}
		case pipeRuntimeRe.MatchString(p):
			runtime = p
		case pipeRatingRe.MatchString(p):
			rating = p
		case pipeDateRe.MatchString(p):
			releaseDate = p
		}
	}
	return
}

func cleanTitle(s string) string {
	return strings.TrimSpace(titleSuffixRe.ReplaceAllString(s, ""))
}

func extractListingDescription(td *goquery.Selection) string {
	clone := td.Clone()
	clone.Find("a, h3, small, font, script").Remove()
	// Genre links are separated by " / " text nodes; removing the <a> elements
	// leaves those separators at the end of the string — strip them.
	return strings.TrimRight(strings.TrimSpace(clone.Text()), " /")
}

func parseYearFromDate(s string) int {
	t, err := time.Parse("January 2, 2006", s)
	if err != nil {
		m := yearFallbackRe.FindString(s)
		if m == "" {
			return 0
		}
		year, _ := strconv.Atoi(m)
		return year
	}
	return t.Year()
}
