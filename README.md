# FIT Lab

Garmin 등에서 내보낸 **FIT 운동 기록 파일**을 업로드해서 분석하고 시각화하는 웹 애플리케이션이다.
백엔드는 **Go**, 프론트엔드는 **React (Vite + TypeScript)** 로 구성한다.

> 🚧 개발 초기 단계다. 아래 로드맵 순서대로 기능을 늘려갈 예정이다.

---

## 목표

- FIT 파일을 업로드하면 활동 요약(거리, 시간, 평균/최대 심박 등)과 초 단위 기록을 추출한다.
- 심박, 페이스(또는 파워), 고도 등을 차트로 보여주고, GPS 경로를 지도에 그린다.
- 여러 활동을 모아 주간 훈련량, 심박존 분포 같은 집계를 제공한다.
- 기기·펌웨어마다 다른 필드를 **공통 내부 모델**로 정규화한다.

## 주요 기능 (계획)

| 단계 | 기능 | 상태 |
| --- | --- | --- |
| 1 | FIT 업로드 → 파싱 → JSON 반환, 심박/페이스 차트 | ⬜ |
| 2 | PostgreSQL 저장, 활동 목록/상세 조회 | ⬜ |
| 3 | 지도 경로 표시, 심박존 분포 | ⬜ |
| 4 | 주간 집계, 훈련 부하, 활동 간 비교 | ⬜ |
| 5 | (선택) FIT 편집/변환, 이상치 탐지 등 | ⬜ |

## 기술 스택

**Backend**
- Go 표준 라이브러리 `net/http` 기반 HTTP API (필요하면 `chi`로 라우팅 확장)
- [`github.com/muktihari/fit`](https://github.com/muktihari/fit): FIT 파싱 후보
- `encoding/json`: JSON 직렬화
- PostgreSQL, `database/sql` 기반 [`pgx`](https://github.com/jackc/pgx) 드라이버
- `net/http` 미들웨어 또는 필요에 따라 별도 패키지 사용

**Frontend**
- React + Vite + TypeScript
- TanStack Query: 데이터 패칭
- 차트: Recharts 또는 uPlot (포인트가 많을 경우)
- 지도: react-leaflet 또는 MapLibre GL

## 프로젝트 구조

```
fit-lab/
├── cmd/server/main.go       # Go 서버 진입점
├── docs/                    # 설계 및 스키마 문서
├── internal/
│   ├── analysis/            # 심박존, 부하, 페이스 계산
│   ├── config/              # 환경 변수 및 설정 로딩
│   ├── fit/                 # FIT 파싱 및 내부 모델 정규화
│   ├── http/handler.go      # API 핸들러와 라우팅
│   └── store/               # 저장소 인터페이스 및 DB 구현
├── migrations/              # DB 마이그레이션
├── samples/                 # 로컬 FIT 파일 (Git 제외)
├── frontend/
│   ├── package.json
│   ├── vite.config.ts
│   └── src/
│       ├── api/             # 백엔드 호출 함수
│       ├── components/      # 재사용 UI 컴포넌트
│       └── pages/           # 화면
├── go.mod
└── README.md
```

## 시작하기

### 사전 준비

- Go (버전은 `go.mod` 기준)
- Node.js 20.19 이상 또는 22.12 이상 (프론트엔드 개발 시)

### 백엔드 실행

1. **환경 변수 설정**: `.env.example`을 복사해 `.env`를 생성하고 DB 접속 정보를 설정한다. (다른 경로는 `ENV_FILE`로 지정)
2. **DB 준비**: PostgreSQL 서버를 구동하거나 SSH 터널을 연결한다.
3. **서버 구동**:

```bash
cp .env.example .env
go run ./cmd/server
# 기본 주소: http://localhost:3000
# 상태 확인: http://localhost:3000/healthz (서버), /readyz (DB 연결)
```

### 프론트엔드 실행

```bash
cd frontend
pnpm install
pnpm dev
# 기본 주소: http://localhost:5173
```

- **테스트**: `pnpm test` (실행), `pnpm test:watch` (감시 모드)
- **개발 프록시**: Vite가 `/api`, `/healthz`, `/readyz` 요청을 백엔드(`localhost:3000`)로 전달한다.

## API 초안

| 메서드 | 경로 | 설명 |
| --- | --- | --- |
| `GET` | `/healthz` | 서버 상태 확인 (Liveness) |
| `GET` | `/readyz` | PostgreSQL 연결 확인 (Readiness) |
| `POST` | `/api/activities` | FIT 파일 수신 (`multipart/form-data`, `file` 필드, 최대 25MB) |
| `GET` | `/api/activities` | 활동 목록 (요약 정보) |
| `GET` | `/api/activities/:id` | 활동 상세 (요약 + 레코드) |
| `GET` | `/api/activities/:id/records?downsample=N` | 초 단위 레코드 (다운샘플링 지원) |
| `DELETE` | `/api/activities/:id` | 활동 삭제 |

### 응답 예시 (활동 요약)

```json
{
  "id": 1,
  "sport": "running",
  "start_time": "2026-10-01T06:30:00Z",
  "duration_sec": 3125,
  "distance_m": 10020.5,
  "avg_hr": 152,
  "max_hr": 178
}
```

## 설계 메모

- **좌표 변환**: FIT semicircle 좌표를 도(degree)로 변환: `semicircles * (180 / 2^31)`
- **정규화 계층**: 기기별 필드 차이를 `Activity`, `Record` 등 공통 내부 모델로 변환 (누락 필드는 nullable 처리)
- **다운샘플링**: 대용량 초 단위 기록 조회를 위한 간격 조정 옵션 제공
- **부하 관리**: 업로드 크기 제한(최대 25MB) 및 요청 타임아웃 적용
- **저장 전략**: 활동 요약과 시계열 레코드를 PostgreSQL에 분리 저장하며, 데이터량에 따라 다운샘플링/압축 저장 검토

## 테스트 데이터

- 본인의 기기에서 내보낸 `.fit` 파일을 `samples/` 폴더에 넣어 사용한다.
- 개인 위치 정보가 포함되어 있으므로 **샘플 파일은 Git에 커밋하지 않는다.** (`.gitignore`에 `samples/` 추가)

## 로드맵

세부 기술 스펙과 검증 계획은 [구현 로드맵 문서](docs/roadmap.md)를 참고한다.

- [ ] 1단계: 업로드 엔드포인트 + 파싱 + 차트 하나
- [ ] 2단계: DB 저장, 활동 목록/상세
- [ ] 3단계: 지도, 심박존 분포
- [ ] 4단계: 주간 집계, 훈련 부하(TRIMP/TSS), 활동 비교
- [ ] 5단계: FIT → GPX/CSV 변환, GPS 튀는 값 보정, 이상치 탐지

## 라이선스

TBD (예: MIT)
