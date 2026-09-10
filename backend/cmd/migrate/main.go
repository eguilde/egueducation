// Command migrate applies the embedded database migrations exactly once per
// deployment. In production it is run only by the Kubernetes/Argo pre-sync
// Job, which is the sole workload given MIGRATION_DATABASE_URL.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/eguilde/egueducation/internal/config"
	"github.com/eguilde/egueducation/internal/db"
)

func main() {
	cfg := config.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync() //nolint:errcheck

	migrationURL, err := cfg.MigrationURL()
	if err != nil {
		logger.Fatal("migration database configuration invalid", zap.Error(err))
	}

	pool, err := db.Open(ctx, migrationURL)
	if err != nil {
		logger.Fatal("migration database connection failed", zap.Error(err))
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		logger.Fatal("database migration failed", zap.Error(err))
	}
	if err := db.ValidateSchemaContract(ctx, pool); err != nil {
		logger.Fatal("schema contract validation failed after migration", zap.Error(err))
	}

	logger.Info("database migrations complete")
}
