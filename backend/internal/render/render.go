// Package render provides HTTP response helpers.
package render

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/OptimusCrime/blu-ray-index/backend/internal/resterr"
)

// JSON writes v as a JSON response. If v is a resterr.Resterr, the appropriate
// HTTP status code and error message are written instead.
func JSON(w http.ResponseWriter, r *http.Request, v any) {
	w.Header().Set("Content-Type", "application/json")

	if re, ok := v.(resterr.Resterr); ok {
		if re.StatusCode >= 500 {
			slog.Error("request error", "err", re.Err, "path", r.URL.Path)
		}
		w.WriteHeader(re.StatusCode)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": re.Err.Error()})
		return
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		slog.Error("failed to encode response", "err", err, "path", r.URL.Path)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(buf.Bytes())
}
