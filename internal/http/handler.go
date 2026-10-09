package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const maxUploadBytes int64 = 25 << 20

type databasePinger interface {
	PingContext(context.Context) error
}

// NewHandler는 API의 최상위 HTTP 핸들러를 생성합니다.
func NewHandler(db databasePinger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /readyz", ready(db))
	mux.HandleFunc("POST /api/activities", uploadActivity)
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func ready(db databasePinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

func uploadActivity(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	file, header, err := r.FormFile("file")
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "업로드 파일은 25MB 이하여야 합니다."})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "multipart/form-data의 file 필드에 FIT 파일을 첨부해 주세요."})
		return
	}
	defer file.Close()

	filename := filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
	if filename == "." || !strings.EqualFold(filepath.Ext(filename), ".fit") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "FIT 파일(.fit)만 업로드할 수 있습니다."})
		return
	}

	contents, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "업로드 파일을 읽지 못했습니다."})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":     "received",
		"filename":   filename,
		"size_bytes": len(contents),
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
