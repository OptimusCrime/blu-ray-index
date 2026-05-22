package bluray

import (
	"context"
	"fmt"
	gohtml "html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
)

const (
	baseListingURL = "https://www.blu-ray.com/movies/movies.php?show=newreleases&sortby=releasetimestamp"
	userAgent      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

var (
	// Matches the original title in parentheses from the HTML page <title>.
	// e.g. "Rider on the Rain 4K Blu-ray (Le passager de la pluie)" → "Le passager de la pluie"
	originalTitleRe = regexp.MustCompile(`Blu-ray\s+\(([^)]+)\)$`)

	// Matches "Rated X" anywhere in the subheading grey span text.
	ratedRe = regexp.MustCompile(`Rated\s+(\S+)`)

	// Matches the production year "(NNNN)" suffix in listing title attributes.
	// Group 1 captures the year digits.
	listingYearRe = regexp.MustCompile(`\s*\((\d{4})\)\s*$`)

	// Matches the production year (or range) between pipes in the subheading grey,
	// e.g. "| 1999 |" or "| 1990-1993 |". Group 1 is the start year, group 2 is
	// the optional end year of a range.
	productionYearRe = regexp.MustCompile(`\|\s*(\d{4})(?:-(\d{4}))?\s*\|`)

	// Matches the year appended to the description prefix by the h3 spacer, e.g. " (1996)".
	descPrefixYearRe = regexp.MustCompile(`^\s*\(\d{4}\)\s*`)

	// Fallback used by parseYearFromDate when time.Parse fails.
	yearFallbackRe = regexp.MustCompile(`\b(\d{4})\b`)

	// Common non-title qualifiers that appear in the page <title> parenthetical.
	editionKeywords = []string{
		"steelbook", "limited edition", "collector", "anniversary",
		"special edition", "deluxe edition", "ultimate edition",
		"slipcase", "slip case", "digibook", "mediabook",
		"premiere", "criterion premiere", "complete series",
		"complete season", "complete collection", "the complete",
		"box set", "boxset", "gift set", "restored", "remastered",
		"extended edition", "theatrical cut", "director's cut",
		"standard edition", "4k ultra hd", "blu-ray", "ultra hd",
		"double feature", "triple feature", "double bill",
	}
)

type scraper struct {
	client *http.Client
}

func newScraper() *scraper {
	return &scraper{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *scraper) get(ctx context.Context, url string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bluray: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bluray: GET %s: status %d", url, resp.StatusCode)
	}

	// Wrap the reader with charset detection so that ISO-8859-1 pages are
	// transparently transcoded to UTF-8 before HTML parsing.
	contentType := resp.Header.Get("Content-Type")
	utf8Reader, err := charset.NewReader(resp.Body, contentType)
	if err != nil {
		return nil, fmt.Errorf("bluray: charset %s: %w", url, err)
	}

	doc, err := goquery.NewDocumentFromReader(utf8Reader)
	if err != nil {
		return nil, fmt.Errorf("bluray: parse %s: %w", url, err)
	}
	return doc, nil
}

// Release dates are derived from the preceding h3 date headers, not from each entry directly.
func (s *scraper) fetchListingPage(ctx context.Context, page int) ([]listingEntry, error) {
	url := baseListingURL
	if page > 0 {
		url = fmt.Sprintf("%s&page=%d", baseListingURL, page)
	}

	doc, err := s.get(ctx, url)
	if err != nil {
		return nil, err
	}

	var entries []listingEntry
	currentDate := ""

	// Walk h3 date headers and hoverlink entries in document order.
	doc.Find("h3[style*='222222'], a.hoverlink").Each(func(_ int, sel *goquery.Selection) {
		if goquery.NodeName(sel) == "h3" {
			currentDate = strings.TrimSpace(sel.Text())
			return
		}

		if currentDate == "" {
			return
		}

		productID, _ := sel.Attr("data-productid")
		href, _ := sel.Attr("href")
		titleAttr, _ := sel.Attr("title")

		if productID == "" || href == "" {
			return
		}

		productionYear := 0
		if m := listingYearRe.FindStringSubmatch(titleAttr); len(m) == 2 {
			productionYear, _ = strconv.Atoi(m[1])
		}
		title := strings.TrimSpace(listingYearRe.ReplaceAllString(titleAttr, ""))

		imageURL := ""
		if img := sel.Find("img.cover"); img.Length() > 0 {
			imageURL, _ = img.Attr("src")
		}

		entries = append(entries, listingEntry{
			productID:      productID,
			url:            href,
			title:          title,
			releaseDate:    currentDate,
			productionYear: productionYear,
			imageURL:       imageURL,
		})
	})

	return entries, nil
}

func (s *scraper) fetchDetailPage(ctx context.Context, entry listingEntry) (*Release, error) {
	doc, err := s.get(ctx, entry.url)
	if err != nil {
		return nil, err
	}

	release := &Release{
		ProductID:   entry.productID,
		URL:         entry.url,
		Title:       entry.title,
		ReleaseDate: entry.releaseDate,
		ImageURL:    entry.imageURL,
		Genres:      []string{},
	}

	release.ReleaseYear = parseYearFromDate(entry.releaseDate)
	// Carry forward the production year parsed from the listing title attribute;
	// the detail-page subheading grey may refine it below (handles year ranges).
	release.ProductionYear = entry.productionYear

	// Original title from the HTML <title> tag, e.g.:
	// "Rider on the Rain 4K Blu-ray (Le passager de la pluie)"
	// The parenthetical may contain " | "-separated parts mixing original title
	// and edition info, e.g. "Leák | Standard Edition" or "Standard Edition | Ratu Ilmu Hitam".
	pageTitle := doc.Find("head title").First().Text()
	if m := originalTitleRe.FindStringSubmatch(pageTitle); len(m) == 2 {
		release.OriginalTitle = extractOriginalTitle(m[1])
	}

	// Cover image: prefer the larger front image from the detail page.
	if img := doc.Find("img.coverfront").First(); img.Length() > 0 {
		if src, ok := img.Attr("src"); ok && src != "" {
			release.ImageURL = src
		}
	}

	// Subheading grey: "Studio | Year | Runtime | Rated X | Release Date"
	doc.Find("span.subheading.grey").First().Each(func(_ int, sel *goquery.Selection) {
		sel.Find("a[href*='studioid']").Each(func(_ int, a *goquery.Selection) {
			if release.Studio == "" {
				release.Studio = strings.TrimSpace(a.Text())
			}
		})

		if rt := sel.Find("span#runtime").First(); rt.Length() > 0 {
			release.Runtime = strings.TrimSpace(rt.Text())
		}

		if m := ratedRe.FindStringSubmatch(sel.Text()); len(m) == 2 {
			release.Rating = "Rated " + m[1]
		}

		// Production year or range between pipes (e.g. "| 1999 |" or "| 1990-1993 |").
		// For ranges, use the end year so a recent series finale is not excluded.
		if m := productionYearRe.FindStringSubmatch(sel.Text()); len(m) == 3 {
			yr := m[1]
			if m[2] != "" {
				yr = m[2]
			}
			release.ProductionYear, _ = strconv.Atoi(yr)
		}
	})

	// Genres from genreappeal divs (located in the movie-info column).
	doc.Find("div.genreappeal").Each(func(_ int, sel *goquery.Selection) {
		if genre := strings.TrimSpace(sel.Text()); genre != "" {
			release.Genres = append(release.Genres, genre)
		}
	})

	if info := doc.Find("#movie_info").First(); info.Length() > 0 {
		release.Description = extractDescription(info)
	}

	return release, nil
}

// extractDescription returns the movie synopsis from the #movie_info element.
// The synopsis sits after the h3 title (and optional screenshots table) and
// before the "Director:" attribution line.
func extractDescription(info *goquery.Selection) string {
	// Remove the title h3, any screenshot tables, and screenshot captions so
	// that only the synopsis text and attribution lines remain.
	clone := info.Clone()
	clone.Find("h3, table, center, script").Remove()

	html, err := clone.Html()
	if err != nil {
		return ""
	}

	for _, marker := range []string{"Director:", "Writer:", "Starring:"} {
		if idx := strings.Index(html, marker); idx >= 0 {
			html = html[:idx]
			break
		}
	}

	// The preamble after the h3 removal is: &nbsp;(YEAR)\n<br>...<br></font>\n
	// Find the last </font>\n or </center> — the description starts right after.
	lastFont := strings.LastIndex(html, "</font>")
	lastCenter := strings.LastIndex(html, "</center>")
	cutAt := lastFont
	if lastCenter > cutAt {
		cutAt = lastCenter
	}

	if cutAt >= 0 {
		html = html[cutAt:]
		if nl := strings.Index(html, "\n"); nl >= 0 {
			html = html[nl+1:]
		}
	}

	// Take the first double-br chunk to drop any "For more about…" notices.
	if idx := strings.Index(html, "<br><br>"); idx >= 0 {
		html = html[:idx]
	}

	text := stripTags(html)
	// The remaining text may start with a "(YEAR) " artefact from the &nbsp; node
	// that follows the h3; strip it.
	text = descPrefixYearRe.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)
	return gohtml.UnescapeString(text)
}

func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '<':
			inTag = true
		case c == '>':
			inTag = false
			b.WriteByte(' ')
		case !inTag:
			b.WriteByte(c)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// The parenthetical may be pipe-separated, mixing original title with edition qualifiers (e.g. "Leák | Standard Edition").
func extractOriginalTitle(raw string) string {
	parts := strings.Split(raw, " | ")
	var titleParts []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && !isEditionQualifier(p) {
			titleParts = append(titleParts, p)
		}
	}
	return strings.Join(titleParts, " / ")
}

// isEditionQualifier returns true when s is a common packaging/edition descriptor
// rather than a foreign-language original title.
func isEditionQualifier(s string) bool {
	lower := strings.ToLower(s)
	for _, kw := range editionKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// parseYearFromDate extracts the 4-digit year from a string like "May 12, 2026".
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
