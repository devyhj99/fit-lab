-- PostgreSQL 초기 활동 데이터 스키마 (FIT 및 GPX)
-- 이 파일은 DDL 초안입니다. 백업과 적용 환경을 확인한 뒤 명시적으로 실행하세요.
-- 롤백은 core 스키마 DROP 대신 백업 복원 또는 별도 보정 migration으로 처리합니다.
-- DATABASE_URL이 선택한 fitlab 데이터베이스 안에 core 스키마를 만듭니다. 데이터베이스는 별도 생성합니다.
-- Go 연결은 DATABASE_URL에 search_path=core를 주거나 쿼리에서 core.table을 사용합니다.
-- 상태/범위/형식 검증은 Go 저장 계층에서 수행하며, 이 DDL은 값 CHECK 제약을 두지 않습니다.

CREATE SCHEMA IF NOT EXISTS core;

CREATE TABLE core.activity_files (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_format text NOT NULL, -- 내용 검사 결과: fit 또는 gpx
    sha256 char(64) NOT NULL UNIQUE,
    storage_key text, -- 원본 보관을 끄면 NULL
    original_filename text NOT NULL,
    size_bytes bigint NOT NULL,
    parse_status text NOT NULL,
    parse_error text,
    parser_version text,
    uploaded_at timestamptz NOT NULL DEFAULT now(),
    parsed_at timestamptz
);

CREATE TABLE core.activities (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_file_id bigint NOT NULL REFERENCES core.activity_files(id) ON DELETE RESTRICT,
    file_activity_no integer NOT NULL,
    name text,
    sport text,
    sub_sport text,
    started_at timestamptz,
    utc_offset_seconds integer,
    elapsed_seconds double precision,
    moving_seconds double precision,
    distance_m double precision,
    elevation_gain_m double precision,
    avg_speed_mps double precision,
    max_speed_mps double precision,
    avg_heart_rate_bpm integer,
    max_heart_rate_bpm integer,
    avg_power_w integer,
    max_power_w integer,
    calories integer,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_file_id, file_activity_no)
);

CREATE TABLE core.sessions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    activity_id bigint NOT NULL REFERENCES core.activities(id) ON DELETE CASCADE,
    session_no integer NOT NULL,
    sport text,
    sub_sport text,
    started_at timestamptz,
    elapsed_seconds double precision,
    moving_seconds double precision,
    distance_m double precision,
    elevation_gain_m double precision,
    avg_speed_mps double precision,
    max_speed_mps double precision,
    avg_heart_rate_bpm integer,
    max_heart_rate_bpm integer,
    avg_cadence_rpm double precision,
    avg_power_w integer,
    calories integer,
    UNIQUE (activity_id, session_no),
    UNIQUE (activity_id, id)
);

-- 장치에서 생성된 랩입니다. 사용자 지정/계산 구간은 activity_intervals에 저장합니다.
CREATE TABLE core.laps (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    activity_id bigint NOT NULL REFERENCES core.activities(id) ON DELETE CASCADE,
    session_id bigint,
    lap_no integer NOT NULL,
    start_record_no integer,
    end_record_no integer,
    started_at timestamptz,
    elapsed_seconds double precision,
    moving_seconds double precision,
    distance_m double precision,
    elevation_gain_m double precision,
    avg_speed_mps double precision,
    max_speed_mps double precision,
    avg_heart_rate_bpm integer,
    max_heart_rate_bpm integer,
    avg_cadence_rpm double precision,
    avg_power_w integer,
    calories integer,
    UNIQUE (activity_id, lap_no),
    FOREIGN KEY (activity_id, session_id)
        REFERENCES core.sessions(activity_id, id) ON DELETE CASCADE
);

-- 한 행은 트랙의 표본 하나입니다. 누락되거나 원본에서 제공되지 않은 값은 NULL입니다.
CREATE TABLE core.records (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    activity_id bigint NOT NULL REFERENCES core.activities(id) ON DELETE CASCADE,
    session_id bigint,
    record_no integer NOT NULL,
    recorded_at timestamptz,
    elapsed_seconds double precision,
    distance_m double precision,
    latitude_deg double precision,
    longitude_deg double precision,
    altitude_m double precision,
    speed_mps double precision,
    heart_rate_bpm integer,
    cadence_rpm double precision,
    power_w integer,
    temperature_c double precision,
    extra_fields jsonb NOT NULL DEFAULT '{}'::jsonb, -- FIT 개발자 필드 및 GPX extensions 등
    UNIQUE (activity_id, record_no),
    FOREIGN KEY (activity_id, session_id)
        REFERENCES core.sessions(activity_id, id) ON DELETE CASCADE
);

-- 랩의 레코드 범위가 존재하면 같은 활동의 실제 record를 가리키도록 보장합니다.
ALTER TABLE core.laps
    ADD CONSTRAINT laps_start_record_fk
        FOREIGN KEY (activity_id, start_record_no)
        REFERENCES core.records(activity_id, record_no)
        DEFERRABLE INITIALLY DEFERRED,
    ADD CONSTRAINT laps_end_record_fk
        FOREIGN KEY (activity_id, end_record_no)
        REFERENCES core.records(activity_id, record_no)
        DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE core.activity_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    activity_id bigint NOT NULL REFERENCES core.activities(id) ON DELETE CASCADE,
    event_no integer NOT NULL,
    occurred_at timestamptz,
    elapsed_seconds double precision,
    event_type text NOT NULL,
    event_group text,
    event_data jsonb NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (activity_id, event_no)
);

CREATE INDEX activities_started_at_idx ON core.activities (started_at DESC);
CREATE INDEX activities_sport_started_at_idx ON core.activities (sport, started_at DESC);
CREATE INDEX records_activity_recorded_at_idx ON core.records (activity_id, recorded_at);
CREATE INDEX records_activity_session_record_no_idx
    ON core.records (activity_id, session_id, record_no);
CREATE INDEX laps_activity_session_id_idx ON core.laps (activity_id, session_id);
