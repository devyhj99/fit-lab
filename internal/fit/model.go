package fit

import "time"

// ActivityData는 파싱된 활동 요약과 초 단위 레코드 시계열, 그리고 랩(인터벌) 목록을 담는 컨테이너입니다.
type ActivityData struct {
	Summary ActivitySummary `json:"summary"`
	Records []RecordPoint   `json:"records"`
	Laps    []LapData       `json:"laps,omitempty"`
}

// LapData는 FIT 파일에 기록된 랩(구간/인터벌) 정보입니다.
type LapData struct {
	LapIndex        int       `json:"lap_index"`
	StartTime       time.Time `json:"start_time"`
	DurationSec     float64   `json:"duration_sec"`
	MovingTimeSec   *float64  `json:"moving_time_sec,omitempty"`
	DistanceM       float64   `json:"distance_m"`
	AvgSpeedMps     *float64  `json:"avg_speed_mps,omitempty"`
	MaxSpeedMps     *float64  `json:"max_speed_mps,omitempty"`
	AvgPaceSecPerKm *float64  `json:"avg_pace_sec_per_km,omitempty"`
	AvgHeartRate    *int      `json:"avg_heart_rate,omitempty"`
	MaxHeartRate    *int      `json:"max_heart_rate,omitempty"`
	AvgCadence      *float64  `json:"avg_cadence,omitempty"`
	AvgPower        *int      `json:"avg_power,omitempty"`
	ElevationGainM  *float64  `json:"elevation_gain_m,omitempty"`
	Calories        *int      `json:"calories,omitempty"`
	LapTrigger      string    `json:"lap_trigger,omitempty"`
	Intensity       string    `json:"intensity,omitempty"`
}

// ActivitySummary는 활동 전반의 통계 및 요약 정보입니다.
type ActivitySummary struct {
	WorkoutName     string    `json:"workout_name,omitempty"`
	Sport           string    `json:"sport"`
	SubSport        string    `json:"sub_sport,omitempty"`
	StartTime       time.Time `json:"start_time"`
	DurationSec     float64   `json:"duration_sec"`
	MovingTimeSec   *float64  `json:"moving_time_sec,omitempty"`
	DistanceM       float64   `json:"distance_m"`
	ElevationGainM  *float64  `json:"elevation_gain_m,omitempty"`
	AvgHeartRate    *int      `json:"avg_heart_rate,omitempty"`
	MaxHeartRate    *int      `json:"max_heart_rate,omitempty"`
	AvgSpeedMps     *float64  `json:"avg_speed_mps,omitempty"`
	MaxSpeedMps     *float64  `json:"max_speed_mps,omitempty"`
	AvgPaceSecPerKm *float64  `json:"avg_pace_sec_per_km,omitempty"`
	AvgCadence      *float64  `json:"avg_cadence,omitempty"`
	MaxCadence      *float64  `json:"max_cadence,omitempty"`
	AvgPower        *int      `json:"avg_power,omitempty"`
	MaxPower        *int      `json:"max_power,omitempty"`
	Calories        *int      `json:"calories,omitempty"`
}

// RecordPoint는 초 단위 시계열 표본 하나를 나타냅니다.
type RecordPoint struct {
	Time         time.Time `json:"time"`
	ElapsedSec   float64   `json:"elapsed_sec"`
	DistanceM    *float64  `json:"distance_m,omitempty"`
	Latitude     *float64  `json:"latitude,omitempty"`
	Longitude    *float64  `json:"longitude,omitempty"`
	AltitudeM    *float64  `json:"altitude_m,omitempty"`
	SpeedMps     *float64  `json:"speed_mps,omitempty"`
	PaceSecPerKm *float64  `json:"pace_sec_per_km,omitempty"`
	HeartRate    *int      `json:"heart_rate,omitempty"`
	Cadence      *float64  `json:"cadence,omitempty"`
	Power        *int      `json:"power,omitempty"`
	Temperature  *float64  `json:"temperature,omitempty"`
}
