package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadDotEnv는 ENV_FILE이 있으면 해당 파일을, 없으면 현재 디렉터리와 저장소 루트의 .env를 읽습니다.
// 프로세스 환경변수는 .env 파일의 값보다 우선합니다.
func LoadDotEnv() error {
	if envFile := os.Getenv("ENV_FILE"); envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return fmt.Errorf("ENV_FILE을 읽지 못했습니다: %w", err)
		}
		return nil
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("현재 작업 디렉터리를 확인하지 못했습니다: %w", err)
	}

	for dir := workingDir; ; dir = filepath.Dir(dir) {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			if err := godotenv.Load(envPath); err != nil {
				return fmt.Errorf(".env 파일을 읽지 못했습니다: %w", err)
			}
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf(".env 파일을 확인하지 못했습니다: %w", err)
		}

		if isProjectRoot(dir) {
			return nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
	}
}

// isProjectRoot는 주어진 디렉터리에 go.mod 또는 .git 파일이 존재하는지 검사하여 프로젝트 루트 여부를 판단합니다.
func isProjectRoot(dir string) bool {
	for _, marker := range []string{"go.mod", ".git"} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}
