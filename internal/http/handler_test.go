package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fit-lab/internal/fit"
	"fit-lab/internal/store"

	"github.com/muktihari/fit/encoder"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
	"github.com/muktihari/fit/proto"
)

type storedEntry struct {
	id            int64
	sha256        string
	name          string
	filename      string
	sourceChannel string
	data          *fit.ActivityData
}

type fakeRepository struct {
	pingErr   error
	saveErr   error
	listErr   error
	getErr    error
	deleteErr error
	byID      map[int64]*storedEntry
	bySHA     map[string]int64
	nextID    int64
	lastInput *store.SaveActivityInput
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{
		byID:   make(map[int64]*storedEntry),
		bySHA:  make(map[string]int64),
		nextID: 1,
	}
}

func (r *fakeRepository) PingContext(context.Context) error {
	return r.pingErr
}

func (r *fakeRepository) SaveActivity(_ context.Context, input store.SaveActivityInput) (*store.SaveActivityResult, error) {
	if r.saveErr != nil {
		return nil, r.saveErr
	}
	r.lastInput = &input
	if existingID, exists := r.bySHA[input.SHA256]; exists {
		return &store.SaveActivityResult{
			ActivityID:  existingID,
			FileID:      existingID,
			IsDuplicate: true,
		}, nil
	}
	id := r.nextID
	r.nextID++
	name := strings.TrimSpace(input.ActivityName)
	if name == "" {
		hasGPS := false
		for _, pt := range input.Data.Records {
			if pt.Latitude != nil && pt.Longitude != nil {
				hasGPS = true
				break
			}
		}
		name = store.GenerateDefaultActivityName(input.Data.Summary, hasGPS)
	}
	r.bySHA[input.SHA256] = id
	r.byID[id] = &storedEntry{
		id:            id,
		sha256:        input.SHA256,
		name:          name,
		filename:      input.OriginalFilename,
		sourceChannel: input.SourceChannel,
		data:          input.Data,
	}
	return &store.SaveActivityResult{
		ActivityID:  id,
		FileID:      id,
		Name:        name,
		IsDuplicate: false,
	}, nil
}

func (r *fakeRepository) ListActivities(_ context.Context, limit, offset int) (*store.ActivityListResult, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	if limit <= 0 {
		limit = 20
	}
	items := make([]store.ActivityListItem, 0, len(r.byID))
	for id := int64(1); id < r.nextID; id++ {
		entry, ok := r.byID[id]
		if !ok {
			continue
		}
		items = append(items, store.ActivityListItem{
			ID:            entry.id,
			Name:          entry.name,
			Filename:      entry.filename,
			SourceChannel: entry.sourceChannel,
			Sport:         entry.data.Summary.Sport,
			StartTime:     entry.data.Summary.StartTime,
			DurationSec:   entry.data.Summary.DurationSec,
			DistanceM:     entry.data.Summary.DistanceM,
			CreatedAt:     time.Now().UTC(),
		})
	}
	return &store.ActivityListResult{
		Activities: items,
		Total:      len(items),
		Limit:      limit,
		Offset:     offset,
	}, nil
}

func (r *fakeRepository) GetActivity(_ context.Context, id int64, downsample int) (*store.ActivityDetail, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	entry, ok := r.byID[id]
	if !ok {
		return nil, store.ErrActivityNotFound
	}
	records, _ := r.GetActivityRecords(context.Background(), id, downsample)
	return &store.ActivityDetail{
		ID:            entry.id,
		Name:          entry.name,
		Filename:      entry.filename,
		SourceChannel: entry.sourceChannel,
		Summary:       entry.data.Summary,
		Laps:          entry.data.Laps,
		Records:       records,
	}, nil
}

func (r *fakeRepository) GetActivityRecords(_ context.Context, activityID int64, downsample int) ([]fit.RecordPoint, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	entry, ok := r.byID[activityID]
	if !ok {
		return nil, store.ErrActivityNotFound
	}
	if downsample <= 1 {
		return entry.data.Records, nil
	}
	var sampled []fit.RecordPoint
	for idx, rec := range entry.data.Records {
		if idx%downsample == 0 {
			sampled = append(sampled, rec)
		}
	}
	return sampled, nil
}

func (r *fakeRepository) UpdateActivityName(_ context.Context, id int64, name string) error {
	entry, ok := r.byID[id]
	if !ok {
		return store.ErrActivityNotFound
	}
	entry.name = name
	return nil
}

func (r *fakeRepository) DeleteActivity(_ context.Context, id int64) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	entry, ok := r.byID[id]
	if !ok {
		return store.ErrActivityNotFound
	}
	delete(r.bySHA, entry.sha256)
	delete(r.byID, id)
	return nil
}

func TestHealthReturnsOK(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	NewHandler(newFakeRepository()).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("상태 코드 = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); got != "{\"status\":\"ok\"}\n" {
		t.Fatalf("응답 본문 = %q, want health 응답", got)
	}
}

func TestReadyReflectsDatabaseConnection(t *testing.T) {
	tests := []struct {
		name       string
		dbError    error
		wantStatus int
		wantBody   string
	}{
		{name: "연결됨", wantStatus: http.StatusOK, wantBody: "ready"},
		{name: "연결 실패", dbError: context.DeadlineExceeded, wantStatus: http.StatusServiceUnavailable, wantBody: "unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			repo := newFakeRepository()
			repo.pingErr = tt.dbError

			NewHandler(repo).ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("상태 코드 = %d, want %d", recorder.Code, tt.wantStatus)
			}
			var response map[string]string
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("응답 JSON 파싱 실패: %v", err)
			}
			if response["status"] != tt.wantBody {
				t.Fatalf("status = %q, want %q", response["status"], tt.wantBody)
			}
		})
	}
}

func TestUploadActivity(t *testing.T) {
	validFIT := minimalFITBytes(t)

	tests := []struct {
		name       string
		filename   string
		field      string
		data       []byte
		saveErr    error
		wantStatus int
		wantName   string
	}{
		{name: "유효한 FIT 파일 수신·파싱 및 저장", filename: "run.fit", field: "file", data: validFIT, wantStatus: http.StatusCreated, wantName: "run.fit"},
		{name: "경로가 포함된 파일명 정리", filename: `..\\run.fit`, field: "file", data: validFIT, wantStatus: http.StatusCreated, wantName: "run.fit"},
		{name: "손상된 FIT 데이터 거부", filename: "run.fit", field: "file", data: []byte("corrupted"), wantStatus: http.StatusBadRequest},
		{name: "FIT 이외 형식 거부", filename: "run.txt", field: "file", data: validFIT, wantStatus: http.StatusBadRequest},
		{name: "필수 파일 필드 누락", filename: "run.fit", field: "activity", data: validFIT, wantStatus: http.StatusBadRequest},
		{name: "DB 저장 오류 시 500 반환", filename: "run.fit", field: "file", data: validFIT, saveErr: errors.New("db write error"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := newUploadRequest(t, tt.field, tt.filename, tt.data)
			recorder := httptest.NewRecorder()
			repo := newFakeRepository()
			repo.saveErr = tt.saveErr

			NewHandler(repo).ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("상태 코드 = %d, want %d: %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if tt.wantStatus != http.StatusCreated {
				return
			}

			var response struct {
				Status      string `json:"status"`
				ActivityID  int64  `json:"activity_id"`
				IsDuplicate bool   `json:"is_duplicate"`
				Filename    string `json:"filename"`
				SizeBytes   int    `json:"size_bytes"`
				Summary     struct {
					Sport string `json:"sport"`
				} `json:"summary"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("응답 JSON 파싱 실패: %v", err)
			}
			if response.Status != "received" || response.Filename != tt.wantName || response.SizeBytes != len(tt.data) {
				t.Fatalf("예상하지 못한 업로드 응답: %+v", response)
			}
			if response.ActivityID != 1 || response.IsDuplicate {
				t.Fatalf("첫 업로드 activity_id=%d, is_duplicate=%v", response.ActivityID, response.IsDuplicate)
			}
			if response.Summary.Sport != "running" {
				t.Fatalf("summary.sport = %s, want running", response.Summary.Sport)
			}
			if repo.lastInput == nil || len(repo.lastInput.SHA256) != 64 {
				t.Fatalf("SHA256 해시가 올바르게 전달되지 않았습니다: %+v", repo.lastInput)
			}
		})
	}
}

func TestUploadActivityDeduplication(t *testing.T) {
	validFIT := minimalFITBytes(t)
	repo := newFakeRepository()
	handler := NewHandler(repo)

	// 1차 업로드
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, newUploadRequest(t, "file", "run.fit", validFIT))
	if rec1.Code != http.StatusCreated {
		t.Fatalf("1차 업로드 상태 코드 = %d, want 201", rec1.Code)
	}

	// 2차 동일 파일 업로드 (SHA-256 중복)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, newUploadRequest(t, "file", "run-copy.fit", validFIT))
	if rec2.Code != http.StatusCreated {
		t.Fatalf("2차 업로드 상태 코드 = %d, want 201", rec2.Code)
	}

	var res2 struct {
		ActivityID  int64 `json:"activity_id"`
		IsDuplicate bool  `json:"is_duplicate"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &res2); err != nil {
		t.Fatalf("응답 파싱 실패: %v", err)
	}
	if !res2.IsDuplicate || res2.ActivityID != 1 {
		t.Fatalf("중복 업로드 결과 = %+v, want ActivityID=1, IsDuplicate=true", res2)
	}
}

func TestActivityLifecycleCRUD(t *testing.T) {
	validFIT := minimalFITBytesWithRecords(t, 6)
	repo := newFakeRepository()
	handler := NewHandler(repo)

	// 1. 업로드
	uploadRec := httptest.NewRecorder()
	handler.ServeHTTP(uploadRec, newUploadRequest(t, "file", "morning-run.fit", validFIT))
	if uploadRec.Code != http.StatusCreated {
		t.Fatalf("업로드 실패: %d %s", uploadRec.Code, uploadRec.Body.String())
	}

	// 2. 활동 목록 조회 (GET /api/activities)
	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/api/activities?limit=10&offset=0", nil))
	if listRec.Code != http.StatusOK {
		t.Fatalf("목록 조회 실패: %d %s", listRec.Code, listRec.Body.String())
	}
	var listRes store.ActivityListResult
	if err := json.Unmarshal(listRec.Body.Bytes(), &listRes); err != nil {
		t.Fatalf("목록 응답 파싱 실패: %v", err)
	}
	if listRes.Total != 1 || len(listRes.Activities) != 1 || listRes.Activities[0].Name != "저녁 실내 러닝" {
		t.Fatalf("예상하지 못한 목록 응답: %+v", listRes)
	}

	// 2-1. 활동 이름 수정 (PATCH /api/activities/1)
	patchRec := httptest.NewRecorder()
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/activities/1", strings.NewReader(`{"name":"한강 인터벌 훈련"}`))
	patchReq.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("이름 수정 실패: %d %s", patchRec.Code, patchRec.Body.String())
	}

	// 빈 이름 수정 거부 확인 (400)
	badPatchRec := httptest.NewRecorder()
	badPatchReq := httptest.NewRequest(http.MethodPatch, "/api/activities/1", strings.NewReader(`{"name":"   "}`))
	handler.ServeHTTP(badPatchRec, badPatchReq)
	if badPatchRec.Code != http.StatusBadRequest {
		t.Fatalf("빈 이름 수정 상태 코드 = %d, want 400", badPatchRec.Code)
	}

	// 3. 활동 상세 조회 (GET /api/activities/1)
	detailRec := httptest.NewRecorder()
	handler.ServeHTTP(detailRec, httptest.NewRequest(http.MethodGet, "/api/activities/1", nil))
	if detailRec.Code != http.StatusOK {
		t.Fatalf("상세 조회 실패: %d %s", detailRec.Code, detailRec.Body.String())
	}
	var detailRes store.ActivityDetail
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detailRes); err != nil {
		t.Fatalf("상세 응답 파싱 실패: %v", err)
	}
	if detailRes.ID != 1 || detailRes.Name != "한강 인터벌 훈련" || len(detailRes.Records) != 6 {
		t.Fatalf("예상하지 못한 상세 응답: ID=%d, Name=%q, records=%d", detailRes.ID, detailRes.Name, len(detailRes.Records))
	}

	// 4. 레코드 다운샘플링 조회 (GET /api/activities/1/records?downsample=2)
	recordsRec := httptest.NewRecorder()
	handler.ServeHTTP(recordsRec, httptest.NewRequest(http.MethodGet, "/api/activities/1/records?downsample=2", nil))
	if recordsRec.Code != http.StatusOK {
		t.Fatalf("레코드 다운샘플링 조회 실패: %d %s", recordsRec.Code, recordsRec.Body.String())
	}
	var recordsRes struct {
		ActivityID int64             `json:"activity_id"`
		Downsample int               `json:"downsample"`
		Records    []fit.RecordPoint `json:"records"`
	}
	if err := json.Unmarshal(recordsRec.Body.Bytes(), &recordsRes); err != nil {
		t.Fatalf("레코드 응답 파싱 실패: %v", err)
	}
	if recordsRes.Downsample != 2 || len(recordsRes.Records) != 3 {
		t.Fatalf("다운샘플링 결과 = downsample:%d, len:%d, want downsample:2, len:3", recordsRes.Downsample, len(recordsRes.Records))
	}

	// 5. 잘못된 ID 및 없는 ID 조회 검증
	badIDRec := httptest.NewRecorder()
	handler.ServeHTTP(badIDRec, httptest.NewRequest(http.MethodGet, "/api/activities/abc", nil))
	if badIDRec.Code != http.StatusBadRequest {
		t.Fatalf("잘못된 ID 상태 코드 = %d, want 400", badIDRec.Code)
	}

	notFoundRec := httptest.NewRecorder()
	handler.ServeHTTP(notFoundRec, httptest.NewRequest(http.MethodGet, "/api/activities/999", nil))
	if notFoundRec.Code != http.StatusNotFound {
		t.Fatalf("없는 ID 상태 코드 = %d, want 404", notFoundRec.Code)
	}

	// 6. 활동 삭제 (DELETE /api/activities/1)
	delRec := httptest.NewRecorder()
	handler.ServeHTTP(delRec, httptest.NewRequest(http.MethodDelete, "/api/activities/1", nil))
	if delRec.Code != http.StatusOK {
		t.Fatalf("활동 삭제 실패: %d %s", delRec.Code, delRec.Body.String())
	}

	// 이미 삭제된 활동 재삭제 시 404
	delAgainRec := httptest.NewRecorder()
	handler.ServeHTTP(delAgainRec, httptest.NewRequest(http.MethodDelete, "/api/activities/1", nil))
	if delAgainRec.Code != http.StatusNotFound {
		t.Fatalf("재삭제 상태 코드 = %d, want 404", delAgainRec.Code)
	}

	// 7. 삭제 후 동일 파일 재업로드 시 중복으로 막히지 않고 새로 저장됨
	reuploadRec := httptest.NewRecorder()
	handler.ServeHTTP(reuploadRec, newUploadRequest(t, "file", "morning-run.fit", validFIT))
	if reuploadRec.Code != http.StatusCreated {
		t.Fatalf("재업로드 실패: %d", reuploadRec.Code)
	}
	var reuploadRes struct {
		ActivityID  int64 `json:"activity_id"`
		IsDuplicate bool  `json:"is_duplicate"`
	}
	_ = json.Unmarshal(reuploadRec.Body.Bytes(), &reuploadRes)
	if reuploadRes.IsDuplicate {
		t.Fatal("삭제 후 재업로드 시 IsDuplicate는 false여야 합니다")
	}
}

func minimalFITBytes(t *testing.T) []byte {
	return minimalFITBytesWithRecords(t, 0)
}

func minimalFITBytesWithRecords(t *testing.T, recordCount int) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := encoder.New(&buf)
	fileId := mesgdef.NewFileId(nil).SetType(typedef.FileActivity)
	start := time.Date(2026, 10, 3, 11, 0, 0, 0, time.UTC)
	session := mesgdef.NewSession(nil).
		SetSport(typedef.SportRunning).
		SetStartTime(start).
		SetTotalDistance(100000).
		SetTotalElapsedTime(300000)

	messages := []proto.Message{
		fileId.ToMesg(nil),
		session.ToMesg(nil),
	}
	for i := 0; i < recordCount; i++ {
		rec := mesgdef.NewRecord(nil).
			SetTimestamp(start.Add(time.Duration(i) * time.Second)).
			SetDistance(uint32(i * 300)).
			SetHeartRate(uint8(140 + i))
		messages = append(messages, rec.ToMesg(nil))
	}

	fitFile := &proto.FIT{Messages: messages}
	if err := enc.Encode(fitFile); err != nil {
		t.Fatalf("encode minimal fit failed: %v", err)
	}
	return buf.Bytes()
}

func newUploadRequest(t *testing.T, field, filename string, contents []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("multipart 파일 생성 실패: %v", err)
	}
	if _, err := part.Write(contents); err != nil {
		t.Fatalf("multipart 파일 쓰기 실패: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart 본문 생성 실패: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/activities", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
