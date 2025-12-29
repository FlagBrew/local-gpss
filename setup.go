package main

import (
	"context"

	"github.com/FlagBrew/local-gpss/internal/database"
	"github.com/FlagBrew/local-gpss/internal/database/ent"
	"github.com/FlagBrew/local-gpss/internal/gui"
	"github.com/FlagBrew/local-gpss/internal/utils"
	"github.com/apex/log"
)

func setup() context.Context {
	cli.Parse()
	logger = cli.Logger

	ctx, cancel := context.WithCancel(context.Background())
	ctx = log.NewContext(ctx, logger)
	cfg = utils.Setup(ctx, cli.Flags.Mode)
	if cfg.FancyScreen {
		app = gui.New(cfg, false)
		cli.Logger = utils.NewLogger(log.InfoLevel, cli.Debug, app.GetLogOutput())
		logger = cli.Logger
		ctx = log.NewContext(ctx, logger)
		go func() {
			app.Start(false, nil)
			cancel()
		}()
	}

	db = database.New(ctx, &cfg.Database)
	ctx = ent.NewContext(ctx, db)

	database.Migrate(ctx)

	if cfg.Misc.MigrateOriginalDb {
		utils.MigrateOriginalDb(ctx, cfg)
	}

	if cfg.FancyScreen {
		app.SetDb(db)
	}

	return ctx
}
