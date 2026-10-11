# FIT Lab 구현 로드맵

FIT 운동 기록 분석 및 시각화 웹 애플리케이션의 단계별 구현 로드맵과 세부 기술 스펙을 정의한다.

```mermaid
flowchart LR
    M1["1단계: 파싱 & 차트 MVP"] --> M2["2단계: DB 저장 & 활동 관리"]
    M2 --> M3["3단계: 지도 경로 & 심박존"]
    M3 --> M4["4단계: 주간 집계 & 훈련 부하"]
    M4 --> M5["5단계: (선택) 보정 & 변환"]
```

---

## 🏁 1단계: FIT 파싱 엔진 및 시각화 MVP

- **목표**: FIT 파일을 업로드하면 바이너리를 파싱하여 공통 모델로 변환하고, 심박/페이스 차트로 즉시 렌더링한다.

### 백엔드 (`internal/fit`, `internal/http`)
- [`github.com/muktihari/fit`](https://github.com/muktihari/fit) 라이브러리 연동
- FIT 메시지(`FileId`, `Activity`, `Session`, `Lap`, `Record`) 정규화 모델 매핑
- semicircle 좌표의 십진 도(degree) 변환: `semicircles * (180 / 2^31)`
- 단위 변환(m/s → 페이스/시속 등) 및 누락 필드 nullable 처리
- `POST /api/activities` 응답 확장: 파싱된 요약 및 초 단위 레코드 JSON 반환

### 프론트엔드 (`frontend/src`)
- 시계열 차트 라이브러리 도입 (Recharts 또는 uPlot)
- 심박수(bpm), 페이스(min/km), 고도(m) 멀티 라인 차트 컴포넌트
- 파일 업로드 완료 시 수신 데이터 즉시 차트 렌더링

### 검증 기준
- [ ] 샘플 `.fit` 파일 파싱 단위 테스트 통과
- [ ] 손상된 파일 업로드 시 400 에러 처리
- [ ] 브라우저에서 실제 FIT 파일 업로드 후 심박/페이스 곡선 표시 확인

---

## 💾 2단계: PostgreSQL 저장소 연동 및 활동 관리

- **목표**: DDL 스키마에 맞춰 파싱 결과를 데이터베이스에 영속화하고, 활동 목록/상세 CRUD API를 제공한다.

### 백엔드 (`internal/store`, `internal/http`)
- DB 마이그레이션 적용 (`migrations/0001_initial_activity_schema.up.sql`)
- 트랜잭션 기반 일괄 저장: `activity_files` → `activities` → `sessions` → `records` → `laps`
- `records` 대량 삽입(Bulk Insert) 최적화
- 활동 관리 REST API 구현:
  - `GET /api/activities`: 활동 목록 (요약 지표, 페이지네이션)
  - `GET /api/activities/:id`: 활동 상세 (요약, 세션, 랩)
  - `GET /api/activities/:id/records?downsample=N`: 레코드 간격 다운샘플링 조회
  - `DELETE /api/activities/:id`: 활동 및 시계열 레코드 연쇄 삭제

### 프론트엔드 (`frontend/src`)
- TanStack Query(`@tanstack/react-query`) 도입으로 서버 상태 캐싱
- 활동 목록 화면 (카드/테이블 뷰)
- 활동 상세 화면 (다운샘플링 옵션 지원)

### 검증 기준
- [ ] 대량 시계열 레코드 저장 및 다운샘플링 조회 응답 속도 검증
- [ ] 활동 목록, 상세 조회, 삭제 기능의 통합 테스트 통과

---

## 🗺️ 3단계: 지도 경로 시각화 및 심박존 분석

- **목표**: GPS 경로를 인터랙티브 지도에 표시하고 차트와 연동하며, 심박존(Z1~Z5) 체류 분포를 분석한다.

### 백엔드 (`internal/analysis`)
- 심박존 산출 로직 구현 (`internal/analysis/zones.go`): 최대 심박수 기준 5단계 구간 체류 시간 및 비율 계산
- GPS 경로 Bounding Box 및 간소화 폴리라인 산출

### 프론트엔드 (`frontend/src`)
- 지도 라이브러리(Leaflet 또는 MapLibre GL) 도입
- OpenStreetMap 타일 기반 경로 폴리라인 및 시작/종료/랩 마커 표시
- 차트-지도 인터랙션 연동: 차트 호버 시 지도상 해당 위치 마커 동기화
- 심박존 분포 막대/도넛 차트 컴포넌트

### 검증 기준
- [ ] GPS 데이터가 없는 실내 활동의 예외 처리(지도 숨김)
- [ ] 심박존 계산식 단위 테스트 (경계값 및 0초 처리)
- [ ] 차트-지도 호버 동기화 시 렌더링 성능 검증

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
