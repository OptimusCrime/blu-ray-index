// Package bluray provides scraping and serving of Blu-ray release data.
package bluray

// Release represents a single Blu-ray release with all scraped metadata.
type Release struct {
	ProductID      string   `json:"productId"`
	URL            string   `json:"url"`
	Title          string   `json:"title"`
	OriginalTitle  string   `json:"originalTitle,omitempty"`
	ReleaseDate    string   `json:"releaseDate"`
	ReleaseYear    int      `json:"releaseYear"`
	ProductionYear int      `json:"productionYear"`
	Studio         string   `json:"studio"`
	Runtime        string   `json:"runtime"`
	Rating         string   `json:"rating"`
	Description    string   `json:"description"`
	Genres         []string `json:"genres"`
	ImageURL       string   `json:"-"`
	ImageID        string   `json:"imageId,omitempty"`
}

// listingEntry holds the minimal data extracted from the listing page.
type listingEntry struct {
	productID      string
	url            string
	title          string
	releaseDate    string
	productionYear int
	imageURL       string
}
