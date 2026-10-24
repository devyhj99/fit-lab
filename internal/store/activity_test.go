package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"fit-lab/internal/fit"
)

func TestSaveActivityValidation(t *testing.T) {
	ctx := context.Background()
	validSHA := strings.Repeat("a", 64)
	sampleData := &fit.ActivityData{
		Summary: fit.ActivitySummary{
			Sport:       "running",
			StartTime:   time.Date(2026, 10, 3, 11, 0, 0, 0, time.UTC),
			DurationSec: 1800,
			DistanceM:   5000,
		},
	}

	t.Run("초기화되지 않은 Store 거부", func(t *testing.T) {
		var nilStore *Store
		if err := nilStore.PingContext(ctx); err == nil {
			t.Fatal("nil Store에서 PingContext 오류가 필요합니다")
		}
		if _, err := nilStore.SaveActivity(ctx, SaveActivityInput{}); err == nil {
			t.Fatal("nil Store에서 SaveActivity 오류가 필요합니다")
		}
	})

	storeWithNilDB := NewStore(nil)

	t.Run("데이터 누락 거부", func(t *testing.T) {
		_, err := storeWithNilDB.SaveActivity(ctx, SaveActivityInput{
			OriginalFilename: "run.fit",
			SHA256:           validSHA,
			Data:             nil,
		})
		if err == nil {
			t.Fatal("Data가 nil일 때 오류가 필요합니다")
		}
	})

	t.Run("잘못된 SHA-256 해시 거부", func(t *testing.T) {
		rawStore := &Store{db: nil}
		_, err := rawStore.SaveActivity(ctx, SaveActivityInput{
			OriginalFilename: "run.fit",
			SHA256:           "short-hash",
			Data:             sampleData,
		})
		if err == nil {
			t.Fatal("잘못된 SHA-256 길이에 오류가 필요합니다")
		}
	})
}

func TestDetectHasGPS(t *testing.T) {
	lat := 37.5665
	lng := 126.9780

	if detectHasGPS(nil) {
		t.Fatal("빈 레코드에서 has_gps는 false여야 합니다")
	}

	if detectHasGPS([]fit.RecordPoint{{ElapsedSec: 1}}) {
		t.Fatal("좌표가 없는 레코드에서 has_gps는 false여야 합니다")
	}

	if !detectHasGPS([]fit.RecordPoint{{ElapsedSec: 1, Latitude: &lat, Longitude: &lng}}) {
		t.Fatal("유효한 위경도가 있는 레코드에서 has_gps는 true여야 합니다")
	}
}

func TestNullableHelpers(t *testing.T) {
	if nullableString("   ") != nil {
		t.Fatal("공백 문자열은 nil로 변환되어야 합니다")
	}
	if got := nullableString(" running "); got != "running" {
		t.Fatalf("nullableString = %v, want running", got)
	}
	if nullableTime(time.Time{}) != nil {
		t.Fatal("제로 시간은 nil로 변환되어야 합니다")
	}
	now := time.Now().UTC()
	if got := nullableTime(now); got != now {
		t.Fatalf("nullableTime = %v, want %v", got, now)
	}
}

func TestNormalizePaginationAndPace(t *testing.T) {
	limit, offset := normalizePagination(0, -10)
	if limit != 20 || offset != 0 {
		t.Fatalf("normalizePagination(0, -10) = (%d, %d), want (20, 0)", limit, offset)
	}

	limit, offset = normalizePagination(500, 15)
	if limit != 100 || offset != 15 {
		t.Fatalf("normalizePagination(500, 15) = (%d, %d), want (100, 15)", limit, offset)
	}

	spd := 4.0 // 4 m/s -> 250 sec/km
	if pace := computePaceSecPerKm(&spd, 1000, 250); pace == nil || *pace != 250 {
		t.Fatalf("computePaceSecPerKm with speed = %v, want 250", pace)
	}

	if pace := computePaceSecPerKm(nil, 1000, 300); pace == nil || *pace != 300 {
		t.Fatalf("computePaceSecPerKm fallback = %v, want 300", pace)
	}

	if pace := computePaceSecPerKm(nil, 10, 0); pace != nil {
		t.Fatalf("computePaceSecPerKm invalid = %v, want nil", pace)
	}
}

func TestStoreCRUDValidation(t *testing.T) {
	ctx := context.Background()
	var nilStore *Store

	if _, err := nilStore.ListActivities(ctx, 10, 0); err == nil {
		t.Fatal("nil Store에서 ListActivities 오류가 필요합니다")
	}
	if _, err := nilStore.GetActivity(ctx, 1, 1); err == nil {
		t.Fatal("nil Store에서 GetActivity 오류가 필요합니다")
	}
	if _, err := nilStore.GetActivityRecords(ctx, 1, 1); err == nil {
		t.Fatal("nil Store에서 GetActivityRecords 오류가 필요합니다")
	}
	if err := nilStore.DeleteActivity(ctx, 1); err == nil {
		t.Fatal("nil Store에서 DeleteActivity 오류가 필요합니다")
	}
	if err := nilStore.UpdateActivityName(ctx, 1, "새 이름"); err == nil {
		t.Fatal("nil Store에서 UpdateActivityName 오류가 필요합니다")
	}
}

func TestGenerateDefaultActivityName(t *testing.T) {
	tests := []struct {
		name    string
		summary fit.ActivitySummary
		hasGPS  bool
		want    string
	}{
		{
			name: "워크아웃 이름이 있으면 최우선 사용",
			summary: fit.ActivitySummary{
				WorkoutName: "800m x 10 인터벌",
				Sport:       "running",
				StartTime:   time.Date(2026, 10, 3, 11, 8, 29, 0, time.UTC),
			},
			hasGPS: true,
			want:   "800m x 10 인터벌",
		},
		{
			name: "저녁 야외 러닝 (KST 20:08)",
			summary: fit.ActivitySummary{
				Sport:     "running",
				StartTime: time.Date(2026, 10, 3, 11, 8, 29, 0, time.UTC),
			},
			hasGPS: true,
			want:   "저녁 러닝",
		},
		{
			name: "오전 실내 러닝 (KST 07:30, GPS 없음)",
			summary: fit.ActivitySummary{
				Sport:     "running",
				StartTime: time.Date(2026, 10, 2, 22, 30, 0, 0, time.UTC),
			},
			hasGPS: false,
			want:   "오전 실내 러닝",
		},
		{
			name: "오후 트레일 러닝 (KST 14:00)",
			summary: fit.ActivitySummary{
				Sport:     "running",
				SubSport:  "trail",
				StartTime: time.Date(2026, 10, 3, 5, 0, 0, 0, time.UTC),
			},
			hasGPS: true,
			want:   "오후 트레일 러닝",
		},
		{
			name: "새벽 라이딩 (KST 04:00)",
			summary: fit.ActivitySummary{
				Sport:     "cycling",
				StartTime: time.Date(2026, 10, 2, 19, 0, 0, 0, time.UTC),
			},
			hasGPS: true,
			want:   "새벽 라이딩",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateDefaultActivityName(tt.summary, tt.hasGPS); got != tt.want {
				t.Fatalf("GenerateDefaultActivityName() = %q, want %q", got, tt.want)
			}
		})
	}
}


