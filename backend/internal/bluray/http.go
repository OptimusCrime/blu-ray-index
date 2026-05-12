package bluray

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/OptimusCrime/blu-ray-index/backend/internal/render"
	"github.com/OptimusCrime/blu-ray-index/backend/internal/resterr"
)

// Handler serves HTTP requests for Blu-ray release data.
type Handler struct {
	svc    *Service
	imgCli *http.Client
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc:    svc,
		imgCli: &http.Client{Timeout: 15 * time.Second},
	}
}

// Releases handles GET /api/releases?page=N
func (h *Handler) Releases(w http.ResponseWriter, r *http.Request) {
	page := 0
	if p := r.URL.Query().Get("page"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			render.JSON(w, r, resterr.New("invalid page parameter", http.StatusBadRequest))
			return
		}
		page = n
	}

	releases, err := h.svc.Releases(r.Context(), page)
	if err != nil {
		render.JSON(w, r, resterr.FromErr(err, http.StatusInternalServerError))
		return
	}

	render.JSON(w, r, releases)
}

// Image handles GET /api/image?url=... and proxies the image from blu-ray.com.
// This avoids CORS and hotlink restrictions on the frontend.
func (h *Handler) Image(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		http.Error(w, "missing url parameter", http.StatusBadRequest)
		return
	}

	// Only allow proxying images from the known CDN host.
	if !strings.HasPrefix(rawURL, "https://images.static-bluray.com/") {
		http.Error(w, "url not allowed", http.StatusForbidden)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, rawURL, nil)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", "https://www.blu-ray.com/")

	resp, err := h.imgCli.Do(req)
	if err != nil {
		slog.Error("image proxy error", "url", rawURL, "err", err)
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.Copy(w, resp.Body)
}
