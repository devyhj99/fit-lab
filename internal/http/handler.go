package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"fit-lab/internal/fit"
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

// health는 서버의 기본 liveness 상태를 반환합니다.
func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ready는 데이터베이스 연결 가능 여부를 확인하여 서비스 준비 상태(readiness)를 반환합니다.
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

// uploadActivity는 클라이언트로부터 업로드된 FIT 바이너리 파일을 수신하고 파싱하여 요약, 시계열 레코드, 랩 데이터를 반환합니다.
// 최대 25MB의 파일 크기 제한과 .fit 확장자 유효성 검증을 수행합니다.
func uploadActivity(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	file, header, err := r.FormFile("file")
	if err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
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

	activity, err := fit.Parse(bytes.NewReader(contents))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("FIT 파일을 파싱하지 못했습니다: %v", err)})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":     "received",
		"filename":   filename,
		"size_bytes": len(contents),
		"summary":    activity.Summary,
		"records":    activity.Records,
		"laps":       activity.Laps,
	})
}

// writeJSON은 지정된 HTTP 상태 코드와 객체를 JSON 형식으로 응답 바디에 인코딩하여 출력합니다.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
