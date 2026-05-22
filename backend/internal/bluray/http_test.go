package bluray

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReleasesHandlerPageValidation(t *testing.T) {
	h := &Handler{svc: New(), imgCli: &http.Client{}}

	tests := []struct {
		query      string
		wantStatus int
	}{
		{"page=abc", http.StatusBadRequest},
		{"page=-1", http.StatusBadRequest},
		{"page=21", http.StatusBadRequest}, // exceeds maxPage
	}

	for _, tc := range tests {
		req := httptest.NewRequest(http.MethodGet, "/api/releases?"+tc.query, nil)
		w := httptest.NewRecorder()
		h.Releases(w, req)
		if w.Code != tc.wantStatus {
			t.Errorf("query %q: status = %d, want %d", tc.query, w.Code, tc.wantStatus)
		}
	}
}
