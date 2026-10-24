-- PostgreSQL 초기 활동 데이터 스키마 (FIT 및 GPX)
-- DATABASE_URL이 선택한 fitlab 데이터베이스 안에 core 스키마를 만듭니다.
-- Go 연결은 DATABASE_URL에 search_path=core를 주거나 쿼리에서 core.table을 사용합니다.
-- 상태/범위/형식 검증은 Go 저장 계층에서 수행합니다.

CREATE SCHEMA IF NOT EXISTS core;

-- 업로드 또는 수집된 원본 파일 메타데이터 (sha256으로 중복 방지)
CREATE TABLE core.activity_files (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_channel text NOT NULL DEFAULT 'upload', -- upload 또는 dropbox
    source_format text NOT NULL, -- fit 또는 gpx
    sha256 text NOT NULL UNIQUE,
    storage_key text, -- 원본 보관을 끄면 NULL
    original_filename text NOT NULL,
    size_bytes bigint NOT NULL,
    parse_status text NOT NULL,
    parse_error text,
    parser_version text,
    uploaded_at timestamptz NOT NULL DEFAULT now(),
    parsed_at timestamptz
);

-- 단일 운동 활동의 요약 및 통계 정보
CREATE TABLE core.activities (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_file_id bigint NOT NULL REFERENCES core.activity_files(id) ON DELETE CASCADE,
    file_activity_no integer NOT NULL DEFAULT 1,
    name text,
    sport text NOT NULL,
    sub_sport text,
    started_at timestamptz NOT NULL,
    utc_offset_seconds integer,
    has_gps boolean NOT NULL DEFAULT false,
    elapsed_seconds double precision NOT NULL,
    moving_seconds double precision,
    distance_m double precision NOT NULL,
    elevation_gain_m double precision,
    avg_speed_mps double precision,
    max_speed_mps double precision,
    avg_heart_rate_bpm integer,
    max_heart_rate_bpm integer,
    avg_cadence_rpm double precision,
    max_cadence_rpm double precision,
    avg_power_w integer,
    max_power_w integer,
    calories integer,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_file_id, file_activity_no)
);

-- 장치에서 기록된 랩(구간/인터벌) 정보
CREATE TABLE core.laps (
    activity_id bigint NOT NULL REFERENCES core.activities(id) ON DELETE CASCADE,
    lap_no integer NOT NULL,
    started_at timestamptz,
    elapsed_seconds double precision NOT NULL,
    moving_seconds double precision,
    distance_m double precision NOT NULL,
    elevation_gain_m double precision,
    avg_speed_mps double precision,
    max_speed_mps double precision,
    avg_heart_rate_bpm integer,
    max_heart_rate_bpm integer,
    avg_cadence_rpm double precision,
    avg_power_w integer,
    calories integer,
    lap_trigger text, -- manual, distance, time, session_end 등
    intensity text, -- active, rest, warmup, cooldown 등
    PRIMARY KEY (activity_id, lap_no)
);

-- 초 단위 시계열 표본 데이터 (누락되거나 원본에서 제공되지 않은 값은 NULL)
CREATE TABLE core.records (
    activity_id bigint NOT NULL REFERENCES core.activities(id) ON DELETE CASCADE,
    record_no integer NOT NULL,
    recorded_at timestamptz,
    elapsed_seconds double precision NOT NULL,
    distance_m double precision,
    latitude_deg double precision,
    longitude_deg double precision,
    altitude_m double precision,
    speed_mps double precision,
    heart_rate_bpm integer,
    cadence_rpm double precision,
    power_w integer,
    temperature_c double precision,
    PRIMARY KEY (activity_id, record_no)
);

CREATE INDEX activities_started_at_idx ON core.activities (started_at DESC);
CREATE INDEX activities_sport_started_at_idx ON core.activities (sport, started_at DESC);
