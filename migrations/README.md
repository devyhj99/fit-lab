# PostgreSQL 마이그레이션

스키마가 추가되면 순서가 있는 PostgreSQL SQL 마이그레이션 파일을 이 디렉터리에 둡니다. `0001_create_activities.up.sql`처럼 번호와 변경 내용을 파일명에 표시하고, 롤백 정책은 각 마이그레이션과 함께 문서화합니다. API 시작 시 스키마를 자동 변경하지 말고 배포 단계에서 마이그레이션을 별도로 적용합니다.

초기 FIT DDL은 `DATABASE_URL`이 지정한 `fitlab` 데이터베이스에 접속해 실행합니다. 데이터베이스 자체는 별도로 생성하며, DDL이 그 안에 `core` 스키마를 만듭니다. 앱 연결은 `.env`의 `search_path=core`를 사용합니다.

`0001_initial_fit_schema.up.sql`에는 데이터 손실을 일으킬 수 있는 자동 `down` 스크립트를 두지 않습니다. 롤백이 필요하면 백업 복원 또는 데이터 보존을 고려한 새 보정 migration을 사용합니다.
