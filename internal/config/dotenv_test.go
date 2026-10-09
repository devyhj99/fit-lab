package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvFindsRepositoryRootFromChildDirectory(t *testing.T) {
	const key = "FITLAB_DOTENV_ROOT_TEST"
	clearEnvironment(t, key)
	t.Setenv("ENV_FILE", "")

	root := t.TempDir()
	child := filepath.Join(root, "cmd", "server")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("하위 디렉터리 생성 실패: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0o600); err != nil {
		t.Fatalf("go.mod 생성 실패: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte(key+"=loaded-from-root\n"), 0o600); err != nil {
		t.Fatalf(".env 생성 실패: %v", err)
	}

	changeWorkingDirectory(t, child)
	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv() 실패: %v", err)
	}
	if got := os.Getenv(key); got != "loaded-from-root" {
		t.Fatalf("환경변수 값 = %q, want %q", got, "loaded-from-root")
	}
}

func TestLoadDotEnvUsesExplicitPath(t *testing.T) {
	const key = "FITLAB_DOTENV_EXPLICIT_TEST"
	clearEnvironment(t, key)

	dir := t.TempDir()
	envPath := filepath.Join(dir, "local.env")
	if err := os.WriteFile(envPath, []byte(key+"=loaded-from-explicit-path\n"), 0o600); err != nil {
		t.Fatalf("환경 파일 생성 실패: %v", err)
	}
	t.Setenv("ENV_FILE", envPath)

	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv() 실패: %v", err)
	}
	if got := os.Getenv(key); got != "loaded-from-explicit-path" {
		t.Fatalf("환경변수 값 = %q, want %q", got, "loaded-from-explicit-path")
	}
}

func TestLoadDotEnvDoesNotOverrideProcessEnvironment(t *testing.T) {
	const key = "FITLAB_DOTENV_PRECEDENCE_TEST"
	t.Setenv(key, "process-value")

	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte(key+"=file-value\n"), 0o600); err != nil {
		t.Fatalf("환경 파일 생성 실패: %v", err)
	}
	t.Setenv("ENV_FILE", envPath)

	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv() 실패: %v", err)
	}
	if got := os.Getenv(key); got != "process-value" {
		t.Fatalf("환경변수 값 = %q, want process-value", got)
	}
}

func clearEnvironment(t *testing.T, key string) {
	t.Helper()
	previous, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("환경변수 초기화 실패: %v", err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, previous)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func changeWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("현재 디렉터리 확인 실패: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("작업 디렉터리 변경 실패: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})
}
