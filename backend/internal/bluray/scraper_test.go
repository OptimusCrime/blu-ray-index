package bluray

import (
	"testing"
)

func TestParseYearFromDate(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"May 12, 2026", 2026},
		{"January 1, 2024", 2024},
		{"2025", 2025},          // fallback regex
		{"Release: 2023", 2023}, // fallback regex
		{"", 0},
		{"no year here", 0},
	}
	for _, tc := range tests {
		if got := parseYearFromDate(tc.in); got != tc.want {
			t.Errorf("parseYearFromDate(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Butchers Bluff Blu-ray", "Butchers Bluff"},
		{"Road House 4K Blu-ray", "Road House 4K"},
		{"The Mangler 4K Blu-ray", "The Mangler 4K"},
		{"Harry Potter / Fantastic Beasts: 11-Film Collection 4K Blu-ray", "Harry Potter / Fantastic Beasts: 11-Film Collection 4K"},
		{"It's in Our Blood: Red Rockers Blu-ray", "It's in Our Blood: Red Rockers"},
	}
	for _, tc := range tests {
		if got := cleanTitle(tc.in); got != tc.want {
			t.Errorf("cleanTitle(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParsePipeInfo(t *testing.T) {
	tests := []struct {
		in         string
		wantStudio string
		wantDate   string
		wantRT     string
		wantRating string
		wantYear   int
	}{
		{
			in:         "Dark Star Pictures | 2023 | 123 min | Not rated | 2K Blu-ray: Region A (B, C untested) | May 23, 2026",
			wantStudio: "Dark Star Pictures",
			wantDate:   "May 23, 2026",
			wantRT:     "123 min",
			wantRating: "Not rated",
			wantYear:   2023,
		},
		{
			// Year range: use end year; movie count and large runtime are handled
			in:         "Warner Bros. | 2001-2022 | 11 Movies | 1588 min | Rated PG-13 | 4K Blu-ray: Region free | May 19, 2026",
			wantStudio: "Warner Bros.",
			wantDate:   "May 19, 2026",
			wantRT:     "1588 min",
			wantRating: "Rated PG-13",
			wantYear:   2022,
		},
		{
			// No runtime
			in:         "Magnolia Pictures | 2026 | Not rated | 2K Blu-ray: Region A (B, C untested) | May 19, 2026",
			wantStudio: "Magnolia Pictures",
			wantDate:   "May 19, 2026",
			wantRT:     "",
			wantRating: "Not rated",
			wantYear:   2026,
		},
		{
			// No runtime, no rating
			in:         "Music Theories Recordings | 2025 | 2K Blu-ray: Region A (B, C untested) | May 22, 2026",
			wantStudio: "Music Theories Recordings",
			wantDate:   "May 22, 2026",
			wantRT:     "",
			wantRating: "",
			wantYear:   2025,
		},
		{
			// Studio with slash, year range, no runtime
			in:         "Disney / Buena Vista | 2009-2025 | 3 Movies | Rated PG-13 | 2K Blu-ray: Region A (B, C untested) | May 19, 2026",
			wantStudio: "Disney / Buena Vista",
			wantDate:   "May 19, 2026",
			wantRT:     "",
			wantRating: "Rated PG-13",
			wantYear:   2025,
		},
		{
			// Unrated variant
			in:         "Vinegar Syndrome | 1995 | 106 min | Unrated | 4K Blu-ray: Region free | May 22, 2026",
			wantStudio: "Vinegar Syndrome",
			wantDate:   "May 22, 2026",
			wantRT:     "106 min",
			wantRating: "Unrated",
			wantYear:   1995,
		},
	}

	for _, tc := range tests {
		t.Run(tc.in[:20], func(t *testing.T) {
			studio, date, rt, rating, year := parsePipeInfo(tc.in)
			if studio != tc.wantStudio {
				t.Errorf("studio: got %q, want %q", studio, tc.wantStudio)
			}
			if date != tc.wantDate {
				t.Errorf("date: got %q, want %q", date, tc.wantDate)
			}
			if rt != tc.wantRT {
				t.Errorf("runtime: got %q, want %q", rt, tc.wantRT)
			}
			if rating != tc.wantRating {
				t.Errorf("rating: got %q, want %q", rating, tc.wantRating)
			}
			if year != tc.wantYear {
				t.Errorf("year: got %d, want %d", year, tc.wantYear)
			}
		})
	}
}
