package fit

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSampleFIT(t *testing.T) {
	// 실제 샘플 파일 경로
	path := filepath.Join("..", "..", "samples", "2026-10-03-200829-Outdoor Running-WorkOutDoors.fit")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("sample file not found: %v", err)
	}

	activity, err := Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if activity.Summary.Sport != "running" {
		t.Errorf("expected sport running, got %s", activity.Summary.Sport)
	}

	if activity.Summary.DistanceM <= 0 {
		t.Errorf("expected positive distance, got %f", activity.Summary.DistanceM)
	}

	if activity.Summary.DurationSec <= 0 {
		t.Errorf("expected positive duration, got %f", activity.Summary.DurationSec)
	}

	if len(activity.Records) == 0 {
		t.Fatalf("expected non-empty records")
	}

	// 샘플 레코드 검증
	hasGPS := false
	hasHR := false
	for _, pt := range activity.Records {
		if pt.Latitude != nil && pt.Longitude != nil {
			hasGPS = true
		}
		if pt.HeartRate != nil {
			hasHR = true
		}
	}

	if !hasGPS {
		t.Errorf("expected at least one record with valid GPS coordinates")
	}
	if !hasHR {
		t.Errorf("expected at least one record with valid heart rate")
	}

	if len(activity.Laps) == 0 {
		t.Fatalf("expected non-empty laps")
	}
	if activity.Laps[0].LapTrigger == "" {
		t.Errorf("expected lap_trigger to be populated, got empty string")
	}
}

func TestParseInvalidInputs(t *testing.T) {
	t.Run("nil reader", func(t *testing.T) {
		_, err := Parse(nil)
		if err == nil {
			t.Errorf("expected error for nil reader")
		}
	})

	t.Run("corrupted bytes", func(t *testing.T) {
		_, err := Parse(bytes.NewReader([]byte("not a fit file at all")))
		if err == nil {
			t.Errorf("expected error for corrupted fit data")
		}
	})
}
