package main

import (
	"context"

	"github.com/FlagBrew/local-gpss/internal/database"
	"github.com/FlagBrew/local-gpss/internal/database/ent"
	"github.com/FlagBrew/local-gpss/internal/utils"
	"github.com/apex/log"
)

func setup() context.Context {
	cli.Parse()
	logger = cli.Logger

	ctx := log.NewContext(context.Background(), logger)
	cfg = utils.Setup(cli.Flags.Mode)

	db = database.New(ctx, &cfg.Database)
	ctx = ent.NewContext(ctx, db)

	database.Migrate(ctx)

	if cfg.Misc.MigrateOriginalDb {
		utils.MigrateOriginalDb(ctx, cfg)
	}

	return ctx
}
