package http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fit-lab/internal/fit"
	"fit-lab/internal/store"
)

const maxUploadBytes int64 = 25 << 20

// ActivityRepository는 HTTP 계층에서 필요한 데이터베이스 상태 확인 및 활동 CRUD 인터페이스입니다.
type ActivityRepository interface {
	PingContext(context.Context) error
	SaveActivity(context.Context, store.SaveActivityInput) (*store.SaveActivityResult, error)
	ListActivities(ctx context.Context, limit, offset int) (*store.ActivityListResult, error)
	GetActivity(ctx context.Context, id int64, downsample int) (*store.ActivityDetail, error)
	GetActivityRecords(ctx context.Context, activityID int64, downsample int) ([]fit.RecordPoint, error)
	UpdateActivityName(ctx context.Context, id int64, name string) error
	DeleteActivity(ctx context.Context, id int64) error
}

// NewHandler는 API의 최상위 HTTP 핸들러를 생성합니다.
func NewHandler(repo ActivityRepository) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /readyz", ready(repo))
	mux.HandleFunc("POST /api/activities", uploadActivity(repo))
	mux.HandleFunc("GET /api/activities", listActivities(repo))
	mux.HandleFunc("GET /api/activities/{id}", getActivity(repo))
	mux.HandleFunc("GET /api/activities/{id}/records", getActivityRecords(repo))
	mux.HandleFunc("PATCH /api/activities/{id}", updateActivityName(repo))
	mux.HandleFunc("DELETE /api/activities/{id}", deleteActivity(repo))
	return mux
}

// health는 서버의 기본 liveness 상태를 반환합니다.
func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ready는 데이터베이스 연결 가능 여부를 확인하여 서비스 준비 상태(readiness)를 반환합니다.
func ready(repo ActivityRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := repo.PingContext(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

// uploadActivity는 클라이언트로부터 업로드된 FIT 바이너리 파일을 수신·파싱하고 PostgreSQL에 저장한 뒤 결과를 반환합니다.
// 최대 25MB의 파일 크기 제한과 .fit 확장자 유효성 검증, SHA-256 기반 중복 파일 처리를 수행합니다.
func uploadActivity(repo ActivityRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		sum := sha256.Sum256(contents)
		hashHex := hex.EncodeToString(sum[:])

		saveRes, err := repo.SaveActivity(r.Context(), store.SaveActivityInput{
			SourceChannel:    "upload",
			SourceFormat:     "fit",
			OriginalFilename: filename,
			SHA256:           hashHex,
			SizeBytes:        int64(len(contents)),
			Data:             activity,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "활동 데이터를 데이터베이스에 저장하지 못했습니다."})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"status":       "received",
			"activity_id":  saveRes.ActivityID,
			"name":         saveRes.Name,
			"is_duplicate": saveRes.IsDuplicate,
			"filename":     filename,
			"size_bytes":   len(contents),
			"summary":      activity.Summary,
			"records":      activity.Records,
			"laps":         activity.Laps,
		})
	}
}

// listActivities는 저장된 활동 목록을 페이지네이션하여 반환합니다.
func listActivities(repo ActivityRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := parseQueryInt(r, "limit", 20)
		offset := parseQueryInt(r, "offset", 0)

		result, err := repo.ListActivities(r.Context(), limit, offset)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "활동 목록을 불러오지 못했습니다."})
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

// getActivity는 단일 활동의 상세 정보(요약, 랩, 시계열 레코드)를 반환합니다.
func getActivity(repo ActivityRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parsePathID(w, r)
		if !ok {
			return
		}

		downsample := parseQueryInt(r, "downsample", 1)
		detail, err := repo.GetActivity(r.Context(), id, downsample)
		if errors.Is(err, store.ErrActivityNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "요청한 활동을 찾을 수 없습니다."})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "활동 상세 정보를 불러오지 못했습니다."})
			return
		}

		writeJSON(w, http.StatusOK, detail)
	}
}

// getActivityRecords는 특정 활동의 시계열 레코드 목록(다운샘플링 지원)을 반환합니다.
func getActivityRecords(repo ActivityRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parsePathID(w, r)
		if !ok {
			return
		}

		downsample := parseQueryInt(r, "downsample", 1)
		records, err := repo.GetActivityRecords(r.Context(), id, downsample)
		if errors.Is(err, store.ErrActivityNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "요청한 활동을 찾을 수 없습니다."})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "활동 시계열 레코드를 불러오지 못했습니다."})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"activity_id": id,
			"downsample":  max(1, downsample),
			"records":     records,
		})
	}
}

// updateActivityName은 특정 활동의 이름(제목)을 변경합니다.
func updateActivityName(repo ActivityRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parsePathID(w, r)
		if !ok {
			return
		}

		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "올바른 JSON 요청 본문이 필요합니다."})
			return
		}

		trimmed := strings.TrimSpace(body.Name)
		if trimmed == "" || len([]rune(trimmed)) > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "활동 이름은 1자 이상 100자 이하여야 합니다."})
			return
		}

		err := repo.UpdateActivityName(r.Context(), id, trimmed)
		if errors.Is(err, store.ErrActivityNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "수정할 활동을 찾을 수 없습니다."})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "활동 이름을 수정하지 못했습니다."})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":   id,
			"name": trimmed,
		})
	}
}

// deleteActivity는 특정 활동과 연결된 원본 파일·랩·레코드를 삭제합니다.
func deleteActivity(repo ActivityRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parsePathID(w, r)
		if !ok {
			return
		}

		err := repo.DeleteActivity(r.Context(), id)
		if errors.Is(err, store.ErrActivityNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "삭제할 활동을 찾을 수 없습니다."})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "활동을 삭제하지 못했습니다."})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "deleted",
			"id":     id,
		})
	}
}

func parsePathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	rawID := r.PathValue("id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "유효한 활동 ID가 필요합니다."})
		return 0, false
	}
	return id, true
}

func parseQueryInt(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return val
}

// writeJSON은 지정된 HTTP 상태 코드와 객체를 JSON 형식으로 응답 바디에 인코딩하여 출력합니다.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
