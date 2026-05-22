package bluray

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestParseYearFromDate(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"May 12, 2026", 2026},
		{"January 1, 2024", 2024},
		{"2025", 2025},           // fallback regex
		{"Release: 2023", 2023},  // fallback regex
		{"", 0},
		{"no year here", 0},
	}
	for _, tc := range tests {
		if got := parseYearFromDate(tc.in); got != tc.want {
			t.Errorf("parseYearFromDate(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestExtractOriginalTitle(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		// Simple foreign title
		{"Le passager de la pluie", "Le passager de la pluie"},
		// Pipe-separated: qualifier should be dropped
		{"Leák | Standard Edition", "Leák"},
		// Multiple non-qualifier parts joined
		{"Título Original / Alternate", "Título Original / Alternate"},
		// All qualifiers → empty
		{"Steelbook | 4K Ultra HD", ""},
		// Mixed: real title sandwiched between qualifiers
		{"Blu-ray | Título | Steelbook", "Título"},
	}
	for _, tc := range tests {
		if got := extractOriginalTitle(tc.raw); got != tc.want {
			t.Errorf("extractOriginalTitle(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestIsEditionQualifier(t *testing.T) {
	trueCases := []string{"Steelbook", "4K Ultra HD", "Director's Cut", "Box Set", "Remastered"}
	for _, s := range trueCases {
		if !isEditionQualifier(s) {
			t.Errorf("isEditionQualifier(%q) = false, want true", s)
		}
	}

	falseCases := []string{"Le passager de la pluie", "Leák", "Das Boot", "Titulo Original"}
	for _, s := range falseCases {
		if isEditionQualifier(s) {
			t.Errorf("isEditionQualifier(%q) = true, want false", s)
		}
	}
}

func TestStripTags(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"<br>hello<br>", "hello"},
		{"<b>bold</b> text", "bold text"},
		{"no tags", "no tags"},
		{"<a href='x'>link</a> <span>span</span>", "link span"},
	}
	for _, tc := range tests {
		if got := stripTags(tc.in); got != tc.want {
			t.Errorf("stripTags(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "basic synopsis before Director",
			html: `<div id="movie_info"><h3>Title</h3>A story about adventure.<br><br>Director: Jane Doe</div>`,
			want: "A story about adventure.",
		},
		{
			name: "cut at Writer",
			html: `<div id="movie_info"><h3>Title</h3>An epic tale of two cities.<br><br>Writer: Charles Dickens</div>`,
			want: "An epic tale of two cities.",
		},
		{
			name: "strips year prefix artefact",
			html: `<div id="movie_info"><h3>Title</h3>(1999) A noir thriller set in Paris.<br><br>Starring: Jean-Paul</div>`,
			want: "A noir thriller set in Paris.",
		},
		{
			name: "no attribution marker",
			html: `<div id="movie_info"><h3>Title</h3>Simple synopsis with no credits.</div>`,
			want: "Simple synopsis with no credits.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tc.html))
			if err != nil {
				t.Fatalf("failed to parse HTML: %v", err)
			}
			sel := doc.Find("#movie_info").First()
			got := extractDescription(sel)
			if got != tc.want {
				t.Errorf("extractDescription() = %q, want %q", got, tc.want)
			}
		})
	}
}
