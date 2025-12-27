package database

import (
	"context"
	"database/sql"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "modernc.org/sqlite"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/FlagBrew/local-gpss/internal/database/ent"
	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/apex/log"
	"github.com/jackc/pgx/v5/stdlib"
)

func New(ctx context.Context, cfg *models.DatabaseConfig) *ent.Client {
	logger := log.FromContext(ctx)
	var drv *entsql.Driver

	switch cfg.DBType {
	case "postgres":
		poolCfg, err := pgxpool.ParseConfig(cfg.ConnectionString)
		if err != nil {
			logger.WithError(err).Fatal("failed to parse connection string")
			return nil
		}
		pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
		if err != nil {
			logger.WithError(err).Fatal("failed to connect to postgres")
			return nil
		}
		db := stdlib.OpenDBFromPool(pool)
		drv = entsql.OpenDB(dialect.Postgres, db)

		break
	case "mysql":
		db, err := sql.Open(dialect.MySQL, cfg.ConnectionString)
		if err != nil {
			logger.WithError(err).Fatal("failed to connect to mysql")
			return nil
		}

		drv = entsql.OpenDB(dialect.MySQL, db)
		break
	case "sqlite":
		db, err := sql.Open(cfg.DBType, cfg.ConnectionString)
		if err != nil {
			logger.WithError(err).Fatal("failed to connect to sqlite")
			return nil
		}
		drv = entsql.OpenDB(dialect.SQLite, db)
		break
	default:
		return nil
	}

	return ent.NewClient(ent.Driver(drv))
}

func Migrate(ctx context.Context) {
	logger := log.FromContext(ctx)
	logger.Info("initiating database schema migration")
	db := ent.FromContext(ctx)
	if db == nil {
		logger.Fatal("failed to get ent client from context")
		return
	}

	if err := db.Schema.Create(
		ctx,
		schema.WithDropIndex(true),
		schema.WithDropColumn(true),
	); err != nil {
		logger.WithError(err).Fatal("failed to create schema")
	}
	logger.Info("database schema migration complete")
}
