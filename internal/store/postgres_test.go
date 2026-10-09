package store

import (
	"context"
	"testing"
)

func TestOpenPostgresRejectsEmptyDatabaseURL(t *testing.T) {
	db, err := OpenPostgres(context.Background(), " ")
	if err == nil {
		t.Fatal("빈 DATABASE_URL에 오류가 필요합니다")
	}
	if db != nil {
		t.Fatal("연결 풀은 반환되지 않아야 합니다")
	}
}

func TestOpenPostgresRejectsMalformedDatabaseURL(t *testing.T) {
	db, err := OpenPostgres(context.Background(), "postgres://%zz")
	if err == nil {
		t.Fatal("잘못된 DATABASE_URL에 오류가 필요합니다")
	}
	if db != nil {
		t.Fatal("연결 풀은 반환되지 않아야 합니다")
	}
}
