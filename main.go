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

	"go02-neighbor-help/internal/auth"
	"go02-neighbor-help/internal/config"
	"go02-neighbor-help/internal/handler"
	"go02-neighbor-help/internal/server"
	"go02-neighbor-help/internal/service"
	"go02-neighbor-help/internal/store"
	"go02-neighbor-help/internal/web"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log.Printf("config: addr=%s data=%s sessionTTL=%s", cfg.Addr, cfg.DataPath, cfg.SessionTTL)

	fs, err := store.NewFileStore(cfg.DataPath)
	if err != nil {
		return err
	}
	st := fs.Store()
	defer func() { _ = fs.Flush() }()

	hasher := auth.NewPasswordHasher()
	sessions := auth.NewSessionManager(cfg.SessionTTL)
	services := service.NewServices(st, hasher, sessions, nil, cfg.MaxOpenPosts)

	h := handler.New(services, st, sessions, web.Files())
	mux := server.NewMux(h)
	srv := server.New(cfg, mux)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if cfg.SeedAdmin {
		if err := store.SeedAdmin(ctx, st, hasher, cfg.AdminUsername, cfg.AdminPassword); err != nil {
			return err
		}
	}
	if cfg.SeedDemo {
		if err := store.SeedDemo(ctx, st, hasher); err != nil {
			return err
		}
	}

	stopCleaner := startSessionCleaner(sessions)
	defer stopCleaner()

	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		log.Printf("main: received signal %s", sig)
	}

	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel2()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("main: graceful shutdown error: %v", err)
	}
	if err := fs.Flush(); err != nil {
		log.Printf("main: flush error: %v", err)
	}
	log.Println("main: bye")
	return nil
}

func startSessionCleaner(sm *auth.SessionManager) func() {
	ticker := time.NewTicker(10 * time.Minute)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				if n := sm.CleanupExpired(); n > 0 {
					log.Printf("session: cleaned %d expired sessions", n)
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() { close(done) }
}
