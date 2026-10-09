package http

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDatabase struct {
	err error
}

func (db fakeDatabase) PingContext(context.Context) error {
	return db.err
}

func TestHealthReturnsOK(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	NewHandler(fakeDatabase{}).ServeHTTP(recorder, request)

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

			NewHandler(fakeDatabase{err: tt.dbError}).ServeHTTP(recorder, request)

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
	tests := []struct {
		name       string
		filename   string
		field      string
		wantStatus int
		wantName   string
	}{
		{name: "FIT 파일 수신", filename: "run.fit", field: "file", wantStatus: http.StatusCreated, wantName: "run.fit"},
		{name: "경로가 포함된 파일명 정리", filename: `..\\run.fit`, field: "file", wantStatus: http.StatusCreated, wantName: "run.fit"},
		{name: "FIT 이외 형식 거부", filename: "run.txt", field: "file", wantStatus: http.StatusBadRequest},
		{name: "필수 파일 필드 누락", filename: "run.fit", field: "activity", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := newUploadRequest(t, tt.field, tt.filename, []byte("fit-data"))
			recorder := httptest.NewRecorder()

			NewHandler(fakeDatabase{}).ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("상태 코드 = %d, want %d: %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if tt.wantStatus != http.StatusCreated {
				return
			}

			var response struct {
				Status    string `json:"status"`
				Filename  string `json:"filename"`
				SizeBytes int    `json:"size_bytes"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("응답 JSON 파싱 실패: %v", err)
			}
			if response.Status != "received" || response.Filename != tt.wantName || response.SizeBytes != len("fit-data") {
				t.Fatalf("예상하지 못한 업로드 응답: %+v", response)
			}
		})
	}
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
