// Command bpserver is the BalanzaPro3.0 backend entry point.
//
//	bpserver migrate   apply database migrations
//	bpserver serve     start the HTTP server (only /healthz for now) and the
//	                   periodic sweep of expired reservations
//
// Configuration: BP_DATABASE_URL (required), BP_HTTP_ADDR (default
// 127.0.0.1:8080). The inventory HTTP API is not exposed yet: it will be
// defined contract-first in OpenAPI (ADR-009) before any endpoint exists.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/adapters/pgstore"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/inventory/jobs"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/organization"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/postgres"
)

// sweepInterval is how often expired reservations are released.
const sweepInterval = time.Minute

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log, os.Args[1:]); err != nil {
		log.Error("bpserver failed", "error", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger, args []string) error {
	if len(args) != 1 || (args[0] != "migrate" && args[0] != "serve") {
		return errors.New("usage: bpserver migrate|serve")
	}
	url := os.Getenv("BP_DATABASE_URL")
	if url == "" {
		return errors.New("BP_DATABASE_URL is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()

	if args[0] == "migrate" {
		if err := postgres.Migrate(ctx, pool); err != nil {
			return err
		}
		log.Info("migrations applied")
		return nil
	}

	addr := os.Getenv("BP_HTTP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	sweeper := &jobs.ReservationSweeper{
		Expirer:   application.NewService(pgstore.New(pool), nil),
		Companies: organization.NewStore(pool).ListCompanyIDs,
		Interval:  sweepInterval,
		Batch:     500,
		Log:       log,
	}
	go sweeper.Run(ctx)

	srv := &http.Server{Addr: addr, Handler: router(pool), ReadHeaderTimeout: 5 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Info("listening", "addr", addr)
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdown)
	}
}

func router(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer)
	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		if err := pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprint(w, `{"status":"degraded","database":"unreachable"}`)
			return
		}
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	return r
}
