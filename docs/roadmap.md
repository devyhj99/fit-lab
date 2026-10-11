# FIT Lab 구현 로드맵

FIT 운동 기록 분석 및 시각화 웹 애플리케이션의 단계별 구현 로드맵과 세부 기술 스펙을 정의한다.

```mermaid
flowchart LR
    M1["1단계: 파싱 & 차트/지도 MVP"] --> M2["2단계: DB 저장 & 활동 관리"]
    M2 --> M3["3단계: 심박존 & 구간 분석"]
    M3 --> M4["4단계: 주간 집계 & 훈련 부하"]
    M4 --> M5["5단계: (선택) 보정 & 변환"]
```

---

## 🏁 1단계: FIT 파싱 엔진 및 시각화 MVP (차트 + 지도)

- **목표**: FIT 파일을 업로드하면 바이너리를 파싱하여 공통 모델로 변환하고, **uPlot 시계열 차트**와 **Leaflet 지도 경로**, 그리고 **차트-지도 호버 동기화(Hover Sync)**를 즉시 렌더링한다.

### 백엔드 (`internal/fit`, `internal/http`)
- [`github.com/muktihari/fit`](https://github.com/muktihari/fit) 라이브러리 연동
- FIT 메시지(`FileId`, `Activity`, `Session`, `Lap`, `Record`) 정규화 모델 매핑
- semicircle 좌표의 십진 도(degree) 변환: `semicircles * (180 / 2^31)`
- 단위 변환(m/s → 페이스/시속 등) 및 누락 필드 nullable 처리
- `POST /api/activities` 응답 확장: 파싱된 요약 및 초 단위 레코드(위경도, 고도, 심박, 속도 등) JSON 반환

### 프론트엔드 (`frontend/src`)
- **uPlot 시계열 차트**:
  - `uPlot` 도입 (대용량 1초 시계열 고성능 렌더링)
  - 심박수(bpm), 페이스(min/km), 고도(m) 멀티 라인 차트 컴포넌트
- **Leaflet 지도 경로**:
  - `leaflet` + `react-leaflet` 도입 (OpenStreetMap 무료 타일 기반)
  - 운동 경로(GPS Polyline) 렌더링, 시작/종료 지점 마커 표시
- **차트 ↔ 지도 상호작용 동기화 (Hover Sync)**:
  - 차트 커서 이동 시 지도상의 해당 GPS 지점에 마커 핀 실시간 동기화
  - 지도 경로 마우스오버 시 차트 커서 인디케이터 표시
- 파일 업로드 완료 시 수신 데이터로 요약 카드, 차트, 지도 즉시 렌더링

### 검증 기준
- [ ] 샘플 `.fit` 파일 파싱 단위 테스트 통과
- [ ] 손상된 파일 업로드 시 400 에러 처리
- [ ] 브라우저에서 실제 FIT 파일 업로드 후 uPlot 차트 및 Leaflet 경로 렌더링 확인
- [ ] 차트-지도 간 호버 동기화 정상 작동 및 GPS 없는 활동(트레드밀) 시 지도 graceful 숨김 처리

---

## 💾 2단계: PostgreSQL 저장소 연동 및 활동 관리 (수동 업로드 & Dropbox 자동 동기화)

- **목표**: DDL 스키마에 맞춰 파싱 결과를 데이터베이스에 영속화하고, 활동 목록/상세 CRUD API와 **HealthFit → Dropbox Webhook 자동 수집 파이프라인**을 구축한다.

### 백엔드 (`internal/store`, `internal/http`)
- DB 마이그레이션 적용 (`migrations/0001_initial_activity_schema.up.sql`)
- 트랜잭션 기반 일괄 저장: `activity_files` → `activities` → `sessions` → `records` → `laps`
- `records` 대량 삽입(Bulk Insert) 최적화
- **활동 관리 REST API**:
  - `GET /api/activities`: 활동 목록 (요약 지표, 페이지네이션)
  - `GET /api/activities/:id`: 활동 상세 (요약, 세션, 랩)
  - `GET /api/activities/:id/records?downsample=N`: 레코드 간격 다운샘플링 조회
  - `DELETE /api/activities/:id`: 활동 및 시계열 레코드 연쇄 삭제
- **Dropbox Webhook 연동 (HealthFit 자동 수집 파이프라인)**:
  - `GET /api/webhooks/dropbox`: Webhook Challenge 검증 (echo 반환)
  - `POST /api/webhooks/dropbox`: `X-Dropbox-Signature` (HMAC-SHA256) 검증 및 변경 알림 수신
  - 백그라운드 동기화 워커: Dropbox API(`/2/files/list_folder/continue`)로 새로 동기화된 `.fit` 파일 다운로드
  - 파일 파싱 및 DB 적재: 기존 파이프라인을 공유하며 `activity_files.sha256` 유일 제약으로 중복 파일 자동 무시

### 프론트엔드 (`frontend/src`)
- TanStack Query(`@tanstack/react-query`) 도입으로 서버 상태 캐싱
- 활동 목록 화면 (카드/테이블 뷰, 수동 업로드 및 Dropbox 자동 수집 활동 통합 표시)
- 활동 상세 화면 (다운샘플링 옵션 지원)

### 검증 기준
- [ ] 대량 시계열 레코드 저장 및 다운샘플링 조회 응답 속도 검증
- [ ] 활동 목록, 상세 조회, 삭제 기능의 통합 테스트 통과
- [ ] Dropbox Webhook 서명 검증 및 Challenge 핸드셰이크 테스트 통과
- [ ] HealthFit 동기화 파일 수신 시 백그라운드 파싱 및 중복 방지(SHA-256) 적재 검증

---

## 💓 3단계: 심박존 분석 및 구간 통계

- **목표**: 최대 심박수 기준 심박존(Z1~Z5) 체류 분포를 분석하고, 랩(Lap)별 세부 통계를 제공한다.

### 백엔드 (`internal/analysis`)
- 심박존 산출 엔진 (`internal/analysis/zones.go`): 최대/역치 심박수 기준 5단계 구간 체류 시간 및 비율 계산
- 랩(Lap) 구간별 페이스/심박/고도 변화 통계 분석

### 프론트엔드 (`frontend/src`)
- 심박존 체류 분포 막대/도넛 차트 컴포넌트
- 랩(Lap)별 통계 비교 테이블 및 차트 내 랩 분할선(Split Marker) 표시

### 검증 기준
- [ ] 심박존 계산식 단위 테스트 (경계값 및 0초 처리)
- [ ] 랩 데이터 유무에 따른 동적 UI 렌더링 검증

---

## 📊 4단계: 장기 훈련 분석 및 다중 활동 비교

- **목표**: 주간/월간 훈련량을 집계하고, 훈련 부하(TRIMP 등)를 산출하며, 활동 간 기록을 비교 분석한다.

### 백엔드 (`internal/analysis`, `internal/store`)
- 훈련 부하(Banister TRIMP / HRSS) 계산 엔진 구현
- 주간/월간 집계 쿼리 최적화 및 API 구현 (`GET /api/analytics/weekly`)
- 2개 이상 활동 비교 API (`GET /api/activities/compare?ids=...`)

### 프론트엔드 (`frontend/src`)
- 훈련 대시보드 화면 (주간 마일리지 막대 그래프, 피로도/체력 추이)
- 활동 간 오버레이 비교 뷰 (동일 거리 기준 페이스/심박 겹쳐보기)

### 검증 기준
- [ ] 주간 집계 쿼리 성능 및 수치 정확도 검증
- [ ] 서로 다른 길이의 활동 비교 시 정규화 오버레이 차트 렌더링 검증

---

## ⚙️ 5단계 (후속 확장)

- GPX 파서 연동 및 가져오기 지원
- FIT → GPX/CSV 포맷 내보내기/변환
- GPS 튀는 값(스파이크) 필터링 및 고도 보정
- 활동별 파워/페이스 베스트 인터벌 탐지
