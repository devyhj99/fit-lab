package fit

import (
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"time"

	"github.com/muktihari/fit/decoder"
	"github.com/muktihari/fit/profile/mesgdef"
	"github.com/muktihari/fit/profile/typedef"
)

// Parse는 io.Reader로부터 FIT 바이너리 스트림을 읽어 활동 요약과 레코드 목록으로 변환합니다.
func Parse(r io.Reader) (*ActivityData, error) {
	if r == nil {
		return nil, errors.New("reader cannot be nil")
	}

	dec := decoder.New(r)
	fitFile, err := dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("FIT 파일을 디코딩하지 못했습니다: %w", err)
	}

	var session *mesgdef.Session
	var records []*mesgdef.Record
	var laps []*mesgdef.Lap

	for i := range fitFile.Messages {
		mesg := &fitFile.Messages[i]
		switch mesg.Num {
		case typedef.MesgNumSession:
			if session == nil {
				session = mesgdef.NewSession(mesg)
			}
		case typedef.MesgNumRecord:
			records = append(records, mesgdef.NewRecord(mesg))
		case typedef.MesgNumLap:
			laps = append(laps, mesgdef.NewLap(mesg))
		}
	}

	if len(records) == 0 && session == nil {
		return nil, errors.New("활동 레코드나 세션 데이터를 찾을 수 없습니다")
	}

	startTime := time.Time{}
	if session != nil && !session.StartTime.IsZero() {
		startTime = session.StartTime
	} else if len(records) > 0 && !records[0].Timestamp.IsZero() {
		startTime = records[0].Timestamp
	}

	points := make([]RecordPoint, 0, len(records))
	for _, r := range records {
		point := convertRecord(r, startTime)
		points = append(points, point)
	}

	summary := buildSummary(session, points, startTime)
	lapDataList := convertLaps(laps)

	return &ActivityData{
		Summary: summary,
		Records: points,
		Laps:    lapDataList,
	}, nil
}

// convertRecord는 FIT SDK의 레코드 메시지(*mesgdef.Record)를 도메인 모델 RecordPoint로 변환합니다.
// 시작 시간 기준 경과 시간(ElapsedSec)과 심박수, 페이스, GPS, 고도, 케이던스 등의 센서 유효값을 추출합니다.
func convertRecord(r *mesgdef.Record, activityStart time.Time) RecordPoint {
	point := RecordPoint{
		Time: r.Timestamp,
	}

	if !activityStart.IsZero() && !r.Timestamp.IsZero() {
		point.ElapsedSec = r.Timestamp.Sub(activityStart).Seconds()
		if point.ElapsedSec < 0 {
			point.ElapsedSec = 0
		}
	}

	dist := r.DistanceScaled()
	if !math.IsNaN(dist) && dist >= 0 {
		point.DistanceM = &dist
	}

	lat := r.PositionLatDegrees()
	lng := r.PositionLongDegrees()
	if !math.IsNaN(lat) && !math.IsNaN(lng) && (lat != 0 || lng != 0) {
		point.Latitude = &lat
		point.Longitude = &lng
	}

	alt := r.EnhancedAltitudeScaled()
	if math.IsNaN(alt) {
		alt = r.AltitudeScaled()
	}
	if !math.IsNaN(alt) {
		point.AltitudeM = &alt
	}

	spd := r.EnhancedSpeedScaled()
	if math.IsNaN(spd) {
		spd = r.SpeedScaled()
	}
	if !math.IsNaN(spd) && spd >= 0 {
		point.SpeedMps = &spd
		if spd > 0.1 {
			pace := 1000.0 / spd
			point.PaceSecPerKm = &pace
		}
	}

	if r.HeartRate > 0 && r.HeartRate < 255 {
		hr := int(r.HeartRate)
		point.HeartRate = &hr
	}

	if r.Cadence > 0 && r.Cadence < 255 {
		cad := float64(r.Cadence)
		point.Cadence = &cad
	}

	if r.Power > 0 && r.Power < 65535 {
		p := int(r.Power)
		point.Power = &p
	}

	if r.Temperature > -100 && r.Temperature < 100 && r.Temperature != 127 {
		temp := float64(r.Temperature)
		point.Temperature = &temp
	}


	return point
}

// buildSummary는 세션 메시지와 시계열 레코드를 조합하여 활동 전반의 통계 요약(ActivitySummary)을 생성합니다.
// 세션 메시지에 시간이나 거리가 없을 경우 레코드 목록을 역방향 순회하여 누적 거리를 폴백(fallback) 계산합니다.
func buildSummary(session *mesgdef.Session, records []RecordPoint, startTime time.Time) ActivitySummary {
	summary := ActivitySummary{
		Sport:     "generic",
		StartTime: startTime,
	}

	if session != nil {
		if session.Sport != typedef.SportInvalid {
			summary.Sport = session.Sport.String()
		}
		if session.SubSport != typedef.SubSportInvalid {
			summary.SubSport = session.SubSport.String()
		}

		elapsed := session.TotalElapsedTimeScaled()
		if elapsed <= 0 {
			elapsed = session.TotalTimerTimeScaled()
		}
		if elapsed > 0 {
			summary.DurationSec = elapsed
		}

		moving := session.TotalMovingTimeScaled()
		if moving > 0 {
			summary.MovingTimeSec = &moving
		}

		dist := session.TotalDistanceScaled()
		if dist >= 0 {
			summary.DistanceM = dist
		}

		if session.TotalAscent > 0 && session.TotalAscent < 65535 {
			ascent := float64(session.TotalAscent)
			summary.ElevationGainM = &ascent
		}

		if session.AvgHeartRate > 0 && session.AvgHeartRate < 255 {
			avgHR := int(session.AvgHeartRate)
			summary.AvgHeartRate = &avgHR
		}
		if session.MaxHeartRate > 0 && session.MaxHeartRate < 255 {
			maxHR := int(session.MaxHeartRate)
			summary.MaxHeartRate = &maxHR
		}

		avgSpeed := session.AvgSpeedScaled()
		if !math.IsNaN(avgSpeed) && avgSpeed > 0 {
			summary.AvgSpeedMps = &avgSpeed
			if avgSpeed > 0.1 {
				pace := 1000.0 / avgSpeed
				summary.AvgPaceSecPerKm = &pace
			}
		}

		maxSpeed := session.MaxSpeedScaled()
		if !math.IsNaN(maxSpeed) && maxSpeed > 0 {
			summary.MaxSpeedMps = &maxSpeed
		}

		if session.TotalCalories > 0 && session.TotalCalories < 65535 {
			cal := int(session.TotalCalories)
			summary.Calories = &cal
		}
	}

	if summary.DurationSec == 0 && len(records) > 1 {
		summary.DurationSec = records[len(records)-1].ElapsedSec
	}

	if summary.DistanceM == 0 && len(records) > 0 {
		for _, record := range slices.Backward(records) {
			if record.DistanceM != nil {
				summary.DistanceM = *record.DistanceM
				break
			}
		}
	}

	return summary
}

// convertLaps는 FIT SDK의 랩 메시지 목록([]*mesgdef.Lap)을 LapData 슬라이스로 변환합니다.
// 각 랩의 운동 시간, 거리, 평균 페이스, 심박수 및 강도(intensity) 메타데이터를 파싱합니다.
func convertLaps(laps []*mesgdef.Lap) []LapData {
	if len(laps) == 0 {
		return nil
	}

	result := make([]LapData, 0, len(laps))
	for idx, lap := range laps {
		dur := lap.TotalTimerTimeScaled()
		if math.IsNaN(dur) || dur <= 0 {
			dur = lap.TotalElapsedTimeScaled()
		}
		if math.IsNaN(dur) || dur < 0 {
			dur = 0
		}

		dist := lap.TotalDistanceScaled()
		if math.IsNaN(dist) || dist < 0 {
			dist = 0
		}

		item := LapData{
			LapIndex:    idx + 1,
			StartTime:   lap.StartTime,
			DurationSec: dur,
			DistanceM:   dist,
		}

		if lap.Intensity != typedef.IntensityInvalid {
			item.Intensity = lap.Intensity.String()
		}

		if lap.AvgHeartRate > 0 && lap.AvgHeartRate < 255 {
			avgHr := int(lap.AvgHeartRate)
			item.AvgHeartRate = &avgHr
		}

		if lap.MaxHeartRate > 0 && lap.MaxHeartRate < 255 {
			maxHr := int(lap.MaxHeartRate)
			item.MaxHeartRate = &maxHr
		}

		avgSpeed := lap.EnhancedAvgSpeedScaled()
		if math.IsNaN(avgSpeed) || avgSpeed <= 0 {
			avgSpeed = lap.AvgSpeedScaled()
		}
		if !math.IsNaN(avgSpeed) && avgSpeed > 0.1 {
			pace := 1000.0 / avgSpeed
			item.AvgPaceSecPerKm = &pace
		} else if dist > 50 && dur > 0 {
			pace := (dur / dist) * 1000.0
			item.AvgPaceSecPerKm = &pace
		}

		elevGain := float64(lap.TotalAscent)
		if lap.TotalAscent > 0 && lap.TotalAscent < 65535 {
			item.ElevationGainM = &elevGain
		}

		if lap.TotalCalories > 0 && lap.TotalCalories < 65535 {
			cal := int(lap.TotalCalories)
			item.Calories = &cal
		}

		result = append(result, item)
	}

	return result
}
