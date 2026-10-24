package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fit-lab/internal/config"
	api "fit-lab/internal/http"
	"fit-lab/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run은 환경 변수를 로드하고 PostgreSQL 데이터베이스 연결을 검증한 뒤 HTTP 서버를 구동합니다.
// OS 시그널(SIGINT, SIGTERM)을 수신하면 5초 타임아웃의 안전한 graceful shutdown을 수행합니다.
func run() error {
	if err := config.LoadDotEnv(); err != nil {
		return err
	}

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStartup()

	databaseURL := os.Getenv("DATABASE_URL")
	db, err := store.OpenPostgres(startupCtx, databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":3000"
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           api.NewHandler(store.NewStore(db)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("FIT Lab API listening on %s", server.Addr)
		serveErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-serverCtx.Done():
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}

	return nil
}
