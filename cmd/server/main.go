package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"

	"audit-log/cmd/server/wire"
	"audit-log/internal/auditlog/persistence"
	"audit-log/internal/infra/config"
	"audit-log/internal/infra/database"
	"audit-log/internal/infra/telemetry"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Get()
	if err != nil {
		return err
	}
	initLogging(cfg)

	shutdownTelemetry, err := telemetry.Install(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdownTelemetry(ctx); err != nil {
			slog.Error("telemetry shutdown", "err", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var db *gorm.DB
	if cfg.DBDSN == "" {
		slog.Warn("AUDIT_LOG_DB_DSN not set — using in-memory SQLite (data will not persist)")
		db, err = database.OpenInMemory()
		if err != nil {
			return err
		}
		if err := persistence.AutoMigrateModel(db); err != nil {
			return err
		}
	} else {
		if cfg.DBAdminDSN != "" {
			admin, err := database.OpenGORM(cfg.DBAdminDSN)
			if err != nil {
				return err
			}
			sqlAdmin, err := admin.DB()
			if err != nil {
				return err
			}
			defer sqlAdmin.Close()
			if err := persistence.AutoMigrateModel(admin); err != nil {
				return err
			}
			if err := persistence.BootstrapSQL(admin); err != nil {
				slog.Warn("database bootstrap SQL failed (expected on non-Postgres or missing privileges)", "err", err)
			}
		}
		db, err = database.OpenGORM(cfg.DBDSN)
		if err != nil {
			return err
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	grpcSrv, err := wire.InitializeGRPC(db)
	if err != nil {
		return err
	}

	go func() {
		addr := ":" + strconv.Itoa(cfg.PprofPort)
		slog.Info("pprof listening", "addr", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			slog.Error("pprof server", "err", err)
		}
	}()

	lis, err := net.Listen("tcp", ":"+strconv.Itoa(cfg.ServerPort))
	if err != nil {
		return err
	}
	slog.Info("gRPC listening", "addr", lis.Addr().String())

	reflection.Register(grpcSrv)

	errCh := make(chan error, 1)
	go func() {
		if serveErr := grpcSrv.Serve(lis); serveErr != nil {
			errCh <- serveErr
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal")
		stopped := make(chan struct{})
		go func() {
			grpcSrv.GracefulStop()
			close(stopped)
		}()
		select {
		case <-time.After(15 * time.Second):
			grpcSrv.Stop()
		case <-stopped:
		}
		return context.Canceled
	case err := <-errCh:
		return err
	}
}

func initLogging(cfg *config.Config) {
	var h slog.Handler
	opts := &slog.HandlerOptions{Level: levelFromString(cfg.GeneralLogLevel)}
	if cfg.OTelEnvironment == "development" {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
}

func levelFromString(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
