# PostgreSQL 마이그레이션

이 디렉터리는 PostgreSQL 스키마 마이그레이션 SQL 파일을 관리한다.

## 원칙
- **명명 규칙**: `{순번}_{내용}.up.sql` 형식 (예: `0001_initial_activity_schema.up.sql`)
- **실행 방식**: 서버 기동 시 자동 적용하지 않고 배포 단계에서 별도 실행
- **DB 및 스키마**: `fitlab` 데이터베이스의 `core` 스키마 사용 (`.env`의 `search_path=core`)
- **롤백 정책**: 데이터 유실 위험이 있는 자동 down 스크립트 대신 백업 복원 또는 보정 마이그레이션 적용
