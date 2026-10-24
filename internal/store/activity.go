package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"fit-lab/internal/fit"
)

const batchInsertSize = 500

// ErrActivityNotFound는 요청한 활동 ID가 존재하지 않을 때 반환됩니다.
var ErrActivityNotFound = errors.New("활동을 찾을 수 없습니다")

// ActivityListItem은 활동 목록 조회 시 반환되는 개별 활동 요약 항목입니다.
type ActivityListItem struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Filename        string    `json:"filename"`
	SourceChannel   string    `json:"source_channel"`
	Sport           string    `json:"sport"`
	SubSport        string    `json:"sub_sport,omitempty"`
	StartTime       time.Time `json:"start_time"`
	HasGPS          bool      `json:"has_gps"`
	DurationSec     float64   `json:"duration_sec"`
	MovingTimeSec   *float64  `json:"moving_time_sec,omitempty"`
	DistanceM       float64   `json:"distance_m"`
	ElevationGainM  *float64  `json:"elevation_gain_m,omitempty"`
	AvgHeartRate    *int      `json:"avg_heart_rate,omitempty"`
	MaxHeartRate    *int      `json:"max_heart_rate,omitempty"`
	AvgSpeedMps     *float64  `json:"avg_speed_mps,omitempty"`
	AvgPaceSecPerKm *float64  `json:"avg_pace_sec_per_km,omitempty"`
	Calories        *int      `json:"calories,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// ActivityListResult는 페이지네이션이 적용된 활동 목록 조회 결과입니다.
type ActivityListResult struct {
	Activities []ActivityListItem `json:"activities"`
	Total      int                `json:"total"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
}

// ActivityDetail은 단일 활동의 상세 정보(요약, 랩, 시계열 레코드)입니다.
type ActivityDetail struct {
	ID            int64               `json:"id"`
	Name          string              `json:"name"`
	Filename      string              `json:"filename"`
	SourceChannel string              `json:"source_channel"`
	HasGPS        bool                `json:"has_gps"`
	Summary       fit.ActivitySummary `json:"summary"`
	Laps          []fit.LapData       `json:"laps,omitempty"`
	Records       []fit.RecordPoint   `json:"records"`
}

// Store는 PostgreSQL 기반의 활동 데이터 저장소입니다.
type Store struct {
	db *sql.DB
}

// NewStore는 주어진 *sql.DB를 감싸는 Store 인스턴스를 생성합니다.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// PingContext는 데이터베이스 연결이 유효한지 확인합니다.
func (s *Store) PingContext(ctx context.Context) error {
	if s == nil || s.db == nil {
		return errors.New("데이터베이스 연결이 초기화되지 않았습니다")
	}
	return s.db.PingContext(ctx)
}

// SaveActivityInput은 파싱된 활동과 원본 파일 메타데이터를 저장하기 위한 입력 구조체입니다.
type SaveActivityInput struct {
	SourceChannel    string
	SourceFormat     string
	OriginalFilename string
	SHA256           string
	SizeBytes        int64
	ActivityName     string
	Data             *fit.ActivityData
}

// SaveActivityResult는 활동 저장 결과(생성된 ID, 이름 및 중복 여부)를 나타냅니다.
type SaveActivityResult struct {
	ActivityID  int64  `json:"activity_id"`
	FileID      int64  `json:"file_id"`
	Name        string `json:"name"`
	IsDuplicate bool   `json:"is_duplicate"`
}

// SaveActivity는 단일 트랜잭션 안에서 파일 메타데이터(SHA-256 중복 방지), 활동 요약, 랩, 초 단위 레코드를 일괄 저장합니다.
// 이미 동일한 SHA-256 해시의 파일이 존재하면 중복 삽입하지 않고 기존 ID와 IsDuplicate=true를 반환합니다.
func (s *Store) SaveActivity(ctx context.Context, input SaveActivityInput) (*SaveActivityResult, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("데이터베이스 연결이 초기화되지 않았습니다")
	}
	if input.Data == nil {
		return nil, errors.New("저장할 활동 데이터(Data)가 없습니다")
	}
	sha := strings.TrimSpace(strings.ToLower(input.SHA256))
	if len(sha) != 64 {
		return nil, errors.New("유효한 64자리 SHA-256 해시가 필요합니다")
	}
	filename := strings.TrimSpace(input.OriginalFilename)
	if filename == "" {
		return nil, errors.New("원본 파일명(OriginalFilename)이 필요합니다")
	}

	channel := strings.TrimSpace(input.SourceChannel)
	if channel == "" {
		channel = "upload"
	}
	format := strings.TrimSpace(input.SourceFormat)
	if format == "" {
		format = "fit"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("트랜잭션을 시작하지 못했습니다: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var fileID int64
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO core.activity_files (
			source_channel, source_format, sha256, original_filename, size_bytes,
			parse_status, parser_version, parsed_at
		) VALUES ($1, $2, $3, $4, $5, 'parsed', 'v1', now())
		ON CONFLICT (sha256) DO NOTHING
		RETURNING id`,
		channel,
		format,
		sha,
		filename,
		input.SizeBytes,
	).Scan(&fileID)

	if errors.Is(err, sql.ErrNoRows) {
		var (
			existingFileID, existingActivityID int64
			existingName                       string
		)
		lookupErr := tx.QueryRowContext(
			ctx,
			`SELECT af.id, COALESCE(a.id, 0), COALESCE(a.name, '')
			FROM core.activity_files af
			LEFT JOIN core.activities a ON a.source_file_id = af.id AND a.file_activity_no = 1
			WHERE af.sha256 = $1`,
			sha,
		).Scan(&existingFileID, &existingActivityID, &existingName)
		if lookupErr != nil {
			return nil, fmt.Errorf("중복 파일 조회에 실패했습니다: %w", lookupErr)
		}
		if commitErr := tx.Commit(); commitErr != nil {
			return nil, fmt.Errorf("중복 조회 트랜잭션 커밋에 실패했습니다: %w", commitErr)
		}
		return &SaveActivityResult{
			ActivityID:  existingActivityID,
			FileID:      existingFileID,
			Name:        existingName,
			IsDuplicate: true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("파일 메타데이터 저장에 실패했습니다: %w", err)
	}

	summary := input.Data.Summary
	sport := strings.TrimSpace(summary.Sport)
	if sport == "" {
		sport = "generic"
	}

	startedAt := summary.StartTime
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	hasGPS := detectHasGPS(input.Data.Records)

	activityName := strings.TrimSpace(input.ActivityName)
	if activityName == "" {
		activityName = GenerateDefaultActivityName(summary, hasGPS)
	}

	var activityID int64
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO core.activities (
			source_file_id, file_activity_no, name, sport, sub_sport,
			started_at, has_gps, elapsed_seconds, moving_seconds,
			distance_m, elevation_gain_m, avg_speed_mps, max_speed_mps,
			avg_heart_rate_bpm, max_heart_rate_bpm, avg_cadence_rpm, max_cadence_rpm,
			avg_power_w, max_power_w, calories
		) VALUES (
			$1, 1, $2, $3, $4,
			$5, $6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16,
			$17, $18, $19
		)
		RETURNING id`,
		fileID,
		nullableString(activityName),
		sport,
		nullableString(summary.SubSport),
		startedAt,
		hasGPS,
		summary.DurationSec,
		summary.MovingTimeSec,
		summary.DistanceM,
		summary.ElevationGainM,
		summary.AvgSpeedMps,
		summary.MaxSpeedMps,
		summary.AvgHeartRate,
		summary.MaxHeartRate,
		summary.AvgCadence,
		summary.MaxCadence,
		summary.AvgPower,
		summary.MaxPower,
		summary.Calories,
	).Scan(&activityID)
	if err != nil {
		return nil, fmt.Errorf("활동 요약 저장에 실패했습니다: %w", err)
	}

	if err := insertLapsBatch(ctx, tx, activityID, input.Data.Laps); err != nil {
		return nil, err
	}

	if err := insertRecordsBatch(ctx, tx, activityID, input.Data.Records); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("활동 저장 트랜잭션 커밋에 실패했습니다: %w", err)
	}

	return &SaveActivityResult{
		ActivityID:  activityID,
		FileID:      fileID,
		Name:        activityName,
		IsDuplicate: false,
	}, nil
}

func insertLapsBatch(ctx context.Context, tx *sql.Tx, activityID int64, laps []fit.LapData) error {
	if len(laps) == 0 {
		return nil
	}

	const colsPerRow = 16
	for start := 0; start < len(laps); start += batchInsertSize {
		end := min(start+batchInsertSize, len(laps))
		chunk := laps[start:end]

		var query strings.Builder
		query.WriteString(`INSERT INTO core.laps (
			activity_id, lap_no, started_at, elapsed_seconds, moving_seconds,
			distance_m, elevation_gain_m, avg_speed_mps, max_speed_mps,
			avg_heart_rate_bpm, max_heart_rate_bpm, avg_cadence_rpm, avg_power_w,
			calories, lap_trigger, intensity
		) VALUES `)

		args := make([]any, 0, len(chunk)*colsPerRow)
		for i, lap := range chunk {
			if i > 0 {
				query.WriteString(", ")
			}
			base := i * colsPerRow
			fmt.Fprintf(
				&query,
				"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8,
				base+9, base+10, base+11, base+12, base+13, base+14, base+15, base+16,
			)

			lapNo := lap.LapIndex
			if lapNo <= 0 {
				lapNo = start + i + 1
			}

			args = append(
				args,
				activityID,
				lapNo,
				nullableTime(lap.StartTime),
				lap.DurationSec,
				lap.MovingTimeSec,
				lap.DistanceM,
				lap.ElevationGainM,
				lap.AvgSpeedMps,
				lap.MaxSpeedMps,
				lap.AvgHeartRate,
				lap.MaxHeartRate,
				lap.AvgCadence,
				lap.AvgPower,
				lap.Calories,
				nullableString(lap.LapTrigger),
				nullableString(lap.Intensity),
			)
		}

		if _, err := tx.ExecContext(ctx, query.String(), args...); err != nil {
			return fmt.Errorf("랩 데이터 일괄 저장에 실패했습니다: %w", err)
		}
	}

	return nil
}

func insertRecordsBatch(ctx context.Context, tx *sql.Tx, activityID int64, records []fit.RecordPoint) error {
	if len(records) == 0 {
		return nil
	}

	const colsPerRow = 13
	for start := 0; start < len(records); start += batchInsertSize {
		end := min(start+batchInsertSize, len(records))
		chunk := records[start:end]

		var query strings.Builder
		query.WriteString(`INSERT INTO core.records (
			activity_id, record_no, recorded_at, elapsed_seconds,
			distance_m, latitude_deg, longitude_deg, altitude_m,
			speed_mps, heart_rate_bpm, cadence_rpm, power_w, temperature_c
		) VALUES `)

		args := make([]any, 0, len(chunk)*colsPerRow)
		for i, rec := range chunk {
			if i > 0 {
				query.WriteString(", ")
			}
			base := i * colsPerRow
			fmt.Fprintf(
				&query,
				"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				base+1, base+2, base+3, base+4, base+5, base+6, base+7,
				base+8, base+9, base+10, base+11, base+12, base+13,
			)

			recordNo := start + i + 1
			args = append(
				args,
				activityID,
				recordNo,
				nullableTime(rec.Time),
				rec.ElapsedSec,
				rec.DistanceM,
				rec.Latitude,
				rec.Longitude,
				rec.AltitudeM,
				rec.SpeedMps,
				rec.HeartRate,
				rec.Cadence,
				rec.Power,
				rec.Temperature,
			)
		}

		if _, err := tx.ExecContext(ctx, query.String(), args...); err != nil {
			return fmt.Errorf("시계열 레코드 일괄 저장에 실패했습니다: %w", err)
		}
	}

	return nil
}

func detectHasGPS(records []fit.RecordPoint) bool {
	for i := range records {
		if records[i].Latitude != nil && records[i].Longitude != nil {
			return true
		}
	}
	return false
}

func nullableString(s string) any {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// ListActivities는 최신 시작 시각 순으로 활동 목록을 페이지네이션하여 조회합니다.
func (s *Store) ListActivities(ctx context.Context, limit, offset int) (*ActivityListResult, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("데이터베이스 연결이 초기화되지 않았습니다")
	}

	limit, offset = normalizePagination(limit, offset)

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM core.activities`).Scan(&total); err != nil {
		return nil, fmt.Errorf("활동 전체 개수 조회에 실패했습니다: %w", err)
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT
			a.id,
			COALESCE(a.name, ''),
			af.original_filename,
			af.source_channel,
			a.sport,
			COALESCE(a.sub_sport, ''),
			a.started_at,
			a.has_gps,
			a.elapsed_seconds,
			a.moving_seconds,
			a.distance_m,
			a.elevation_gain_m,
			a.avg_heart_rate_bpm,
			a.max_heart_rate_bpm,
			a.avg_speed_mps,
			a.calories,
			a.created_at
		FROM core.activities a
		JOIN core.activity_files af ON af.id = a.source_file_id
		ORDER BY a.started_at DESC, a.id DESC
		LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("활동 목록 조회에 실패했습니다: %w", err)
	}
	defer rows.Close()

	items := make([]ActivityListItem, 0)
	for rows.Next() {
		var item ActivityListItem
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Filename,
			&item.SourceChannel,
			&item.Sport,
			&item.SubSport,
			&item.StartTime,
			&item.HasGPS,
			&item.DurationSec,
			&item.MovingTimeSec,
			&item.DistanceM,
			&item.ElevationGainM,
			&item.AvgHeartRate,
			&item.MaxHeartRate,
			&item.AvgSpeedMps,
			&item.Calories,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("활동 목록 행 읽기에 실패했습니다: %w", err)
		}
		item.AvgPaceSecPerKm = computePaceSecPerKm(item.AvgSpeedMps, item.DistanceM, item.DurationSec)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("활동 목록 순회 중 오류가 발생했습니다: %w", err)
	}

	return &ActivityListResult{
		Activities: items,
		Total:      total,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

// GetActivity는 활동 ID로 요약, 랩 목록, 그리고 다운샘플링된 시계열 레코드를 함께 조회합니다.
func (s *Store) GetActivity(ctx context.Context, id int64, downsample int) (*ActivityDetail, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("데이터베이스 연결이 초기화되지 않았습니다")
	}
	if id <= 0 {
		return nil, ErrActivityNotFound
	}

	var detail ActivityDetail
	err := s.db.QueryRowContext(
		ctx,
		`SELECT
			a.id,
			COALESCE(a.name, ''),
			af.original_filename,
			af.source_channel,
			a.sport,
			COALESCE(a.sub_sport, ''),
			a.started_at,
			a.has_gps,
			a.elapsed_seconds,
			a.moving_seconds,
			a.distance_m,
			a.elevation_gain_m,
			a.avg_heart_rate_bpm,
			a.max_heart_rate_bpm,
			a.avg_speed_mps,
			a.max_speed_mps,
			a.avg_cadence_rpm,
			a.max_cadence_rpm,
			a.avg_power_w,
			a.max_power_w,
			a.calories
		FROM core.activities a
		JOIN core.activity_files af ON af.id = a.source_file_id
		WHERE a.id = $1`,
		id,
	).Scan(
		&detail.ID,
		&detail.Name,
		&detail.Filename,
		&detail.SourceChannel,
		&detail.Summary.Sport,
		&detail.Summary.SubSport,
		&detail.Summary.StartTime,
		&detail.HasGPS,
		&detail.Summary.DurationSec,
		&detail.Summary.MovingTimeSec,
		&detail.Summary.DistanceM,
		&detail.Summary.ElevationGainM,
		&detail.Summary.AvgHeartRate,
		&detail.Summary.MaxHeartRate,
		&detail.Summary.AvgSpeedMps,
		&detail.Summary.MaxSpeedMps,
		&detail.Summary.AvgCadence,
		&detail.Summary.MaxCadence,
		&detail.Summary.AvgPower,
		&detail.Summary.MaxPower,
		&detail.Summary.Calories,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrActivityNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("활동 상세 조회에 실패했습니다: %w", err)
	}

	detail.Summary.AvgPaceSecPerKm = computePaceSecPerKm(
		detail.Summary.AvgSpeedMps,
		detail.Summary.DistanceM,
		detail.Summary.DurationSec,
	)

	laps, err := s.queryLaps(ctx, id)
	if err != nil {
		return nil, err
	}
	detail.Laps = laps

	records, err := s.GetActivityRecords(ctx, id, downsample)
	if err != nil {
		return nil, err
	}
	detail.Records = records

	return &detail, nil
}

// GetActivityRecords는 특정 활동의 시계열 레코드를 조회합니다.
// downsample이 2 이상이면 (record_no - 1) % downsample = 0 조건으로 간격 추출합니다.
func (s *Store) GetActivityRecords(ctx context.Context, activityID int64, downsample int) ([]fit.RecordPoint, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("데이터베이스 연결이 초기화되지 않았습니다")
	}
	if activityID <= 0 {
		return nil, ErrActivityNotFound
	}

	var exists bool
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM core.activities WHERE id = $1)`,
		activityID,
	).Scan(&exists); err != nil {
		return nil, fmt.Errorf("활동 존재 여부 확인에 실패했습니다: %w", err)
	}
	if !exists {
		return nil, ErrActivityNotFound
	}

	var (
		rows *sql.Rows
		err  error
	)
	if downsample > 1 {
		rows, err = s.db.QueryContext(
			ctx,
			`SELECT
				recorded_at, elapsed_seconds, distance_m,
				latitude_deg, longitude_deg, altitude_m,
				speed_mps, heart_rate_bpm, cadence_rpm, power_w, temperature_c
			FROM core.records
			WHERE activity_id = $1 AND ((record_no - 1) % $2 = 0)
			ORDER BY record_no ASC`,
			activityID,
			downsample,
		)
	} else {
		rows, err = s.db.QueryContext(
			ctx,
			`SELECT
				recorded_at, elapsed_seconds, distance_m,
				latitude_deg, longitude_deg, altitude_m,
				speed_mps, heart_rate_bpm, cadence_rpm, power_w, temperature_c
			FROM core.records
			WHERE activity_id = $1
			ORDER BY record_no ASC`,
			activityID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("시계열 레코드 조회에 실패했습니다: %w", err)
	}
	defer rows.Close()

	records := make([]fit.RecordPoint, 0)
	for rows.Next() {
		var (
			rec        fit.RecordPoint
			recordedAt sql.NullTime
		)
		if err := rows.Scan(
			&recordedAt,
			&rec.ElapsedSec,
			&rec.DistanceM,
			&rec.Latitude,
			&rec.Longitude,
			&rec.AltitudeM,
			&rec.SpeedMps,
			&rec.HeartRate,
			&rec.Cadence,
			&rec.Power,
			&rec.Temperature,
		); err != nil {
			return nil, fmt.Errorf("시계열 레코드 행 읽기에 실패했습니다: %w", err)
		}
		if recordedAt.Valid {
			rec.Time = recordedAt.Time
		}
		if rec.SpeedMps != nil && *rec.SpeedMps > 0.1 {
			pace := 1000.0 / *rec.SpeedMps
			rec.PaceSecPerKm = &pace
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("시계열 레코드 순회 중 오류가 발생했습니다: %w", err)
	}

	return records, nil
}

// DeleteActivity는 활동과 연결된 원본 파일 행(activity_files)을 삭제하여 활동, 랩, 레코드를 연쇄 삭제합니다.
func (s *Store) DeleteActivity(ctx context.Context, id int64) error {
	if s == nil || s.db == nil {
		return errors.New("데이터베이스 연결이 초기화되지 않았습니다")
	}
	if id <= 0 {
		return ErrActivityNotFound
	}

	res, err := s.db.ExecContext(
		ctx,
		`DELETE FROM core.activity_files
		WHERE id = (SELECT source_file_id FROM core.activities WHERE id = $1)`,
		id,
	)
	if err != nil {
		return fmt.Errorf("활동 삭제에 실패했습니다: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("삭제된 행 수 확인에 실패했습니다: %w", err)
	}
	if affected == 0 {
		return ErrActivityNotFound
	}
	return nil
}

// UpdateActivityName은 특정 활동의 이름(제목)을 수정합니다.
func (s *Store) UpdateActivityName(ctx context.Context, id int64, name string) error {
	if s == nil || s.db == nil {
		return errors.New("데이터베이스 연결이 초기화되지 않았습니다")
	}
	if id <= 0 {
		return ErrActivityNotFound
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("활동 이름은 비어 있을 수 없습니다")
	}

	res, err := s.db.ExecContext(
		ctx,
		`UPDATE core.activities SET name = $1 WHERE id = $2`,
		trimmed,
		id,
	)
	if err != nil {
		return fmt.Errorf("활동 이름 수정에 실패했습니다: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("수정된 행 수 확인에 실패했습니다: %w", err)
	}
	if affected == 0 {
		return ErrActivityNotFound
	}
	return nil
}

// GenerateDefaultActivityName은 FIT 워크아웃 이름이 있으면 우선 사용하고,
// 없으면 시작 시각(KST 기준 시간대)과 운동 종목/GPS 유무를 조합해 자연스러운 한국어 제목을 생성합니다.
func GenerateDefaultActivityName(summary fit.ActivitySummary, hasGPS bool) string {
	if wkt := strings.TrimSpace(summary.WorkoutName); wkt != "" {
		return wkt
	}

	kst := time.FixedZone("KST", 9*3600)
	start := summary.StartTime
	if start.IsZero() {
		start = time.Now().UTC()
	}
	hour := start.In(kst).Hour()

	var period string
	switch {
	case hour < 5:
		period = "새벽"
	case hour < 11:
		period = "오전"
	case hour < 17:
		period = "오후"
	case hour < 21:
		period = "저녁"
	default:
		period = "야간"
	}

	sportLower := strings.ToLower(strings.TrimSpace(summary.Sport))
	subLower := strings.ToLower(strings.TrimSpace(summary.SubSport))

	var sportLabel string
	switch sportLower {
	case "running":
		switch {
		case strings.Contains(subLower, "treadmill") || strings.Contains(subLower, "indoor") || !hasGPS:
			sportLabel = "실내 러닝"
		case strings.Contains(subLower, "trail"):
			sportLabel = "트레일 러닝"
		case strings.Contains(subLower, "track"):
			sportLabel = "트랙 러닝"
		default:
			sportLabel = "러닝"
		}
	case "cycling", "biking":
		if strings.Contains(subLower, "indoor") || strings.Contains(subLower, "virtual") || !hasGPS {
			sportLabel = "실내 라이딩"
		} else {
			sportLabel = "라이딩"
		}
	case "walking":
		sportLabel = "산책"
	case "hiking":
		sportLabel = "하이킹"
	case "swimming":
		sportLabel = "수영"
	default:
		sportLabel = "운동"
	}

	return fmt.Sprintf("%s %s", period, sportLabel)
}

func (s *Store) queryLaps(ctx context.Context, activityID int64) ([]fit.LapData, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT
			lap_no, started_at, elapsed_seconds, moving_seconds,
			distance_m, elevation_gain_m, avg_speed_mps, max_speed_mps,
			avg_heart_rate_bpm, max_heart_rate_bpm, avg_cadence_rpm, avg_power_w,
			calories, COALESCE(lap_trigger, ''), COALESCE(intensity, '')
		FROM core.laps
		WHERE activity_id = $1
		ORDER BY lap_no ASC`,
		activityID,
	)
	if err != nil {
		return nil, fmt.Errorf("랩 목록 조회에 실패했습니다: %w", err)
	}
	defer rows.Close()

	var laps []fit.LapData
	for rows.Next() {
		var (
			lap       fit.LapData
			startedAt sql.NullTime
		)
		if err := rows.Scan(
			&lap.LapIndex,
			&startedAt,
			&lap.DurationSec,
			&lap.MovingTimeSec,
			&lap.DistanceM,
			&lap.ElevationGainM,
			&lap.AvgSpeedMps,
			&lap.MaxSpeedMps,
			&lap.AvgHeartRate,
			&lap.MaxHeartRate,
			&lap.AvgCadence,
			&lap.AvgPower,
			&lap.Calories,
			&lap.LapTrigger,
			&lap.Intensity,
		); err != nil {
			return nil, fmt.Errorf("랩 행 읽기에 실패했습니다: %w", err)
		}
		if startedAt.Valid {
			lap.StartTime = startedAt.Time
		}
		lap.AvgPaceSecPerKm = computePaceSecPerKm(lap.AvgSpeedMps, lap.DistanceM, lap.DurationSec)
		laps = append(laps, lap)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("랩 순회 중 오류가 발생했습니다: %w", err)
	}

	return laps, nil
}

func normalizePagination(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func computePaceSecPerKm(avgSpeedMps *float64, distanceM, durationSec float64) *float64 {
	if avgSpeedMps != nil && *avgSpeedMps > 0.1 {
		pace := 1000.0 / *avgSpeedMps
		return &pace
	}
	if distanceM > 50 && durationSec > 0 {
		pace := (durationSec / distanceM) * 1000.0
		return &pace
	}
	return nil
}
