package bluray

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/OptimusCrime/blu-ray-index/backend/internal/render"
	"github.com/OptimusCrime/blu-ray-index/backend/internal/resterr"
)

type Handler struct {
	svc    *Service
	imgCli *http.Client
}

func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc:    svc,
		imgCli: &http.Client{Timeout: 15 * time.Second},
	}
}

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

	if page > maxPage {
		render.JSON(w, r, resterr.New(fmt.Sprintf("page %d exceeds maximum allowed page %d", page, maxPage), http.StatusBadRequest))
		return
	}

	releases, err := h.svc.Releases(r.Context(), page)
	if err != nil {
		render.JSON(w, r, resterr.FromErr(err, http.StatusInternalServerError))
		return
	}

	render.JSON(w, r, releases)
}

// The hex ID is resolved to an upstream URL via the image cache so clients never see the raw upstream URL.
func (h *Handler) CoverImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	rawURL, ok := h.svc.ResolveImage(id)
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
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
		slog.Error("image proxy error", "id", id, "err", err)
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
