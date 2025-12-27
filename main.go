package main

import (
	"github.com/FlagBrew/local-gpss/internal/database/ent"
	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/apex/log"
	"github.com/lrstanley/chix"
	"github.com/lrstanley/clix"
)

var (
	cli    = &clix.CLI[models.Flags]{}
	logger log.Interface
	db     *ent.Client
	cfg    *models.Config
)

func main() {
	ctx := setup()

	logger.Infof("Starting HTTP server on %s:%d", cfg.HTTP.ListeningAddr, cfg.HTTP.Port)
	if err := chix.RunContext(ctx, httpServer(ctx)); err != nil {
		db.Close()
	}
}
