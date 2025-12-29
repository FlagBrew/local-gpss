package main

import (
	"fmt"

	"github.com/FlagBrew/local-gpss/internal/database/ent"
	"github.com/FlagBrew/local-gpss/internal/gui"
	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/FlagBrew/local-gpss/internal/utils"
	"github.com/apex/log"
	"github.com/lrstanley/chix"
	"github.com/lrstanley/clix"
	_ "modernc.org/sqlite"
)

var (
	cli    = &clix.CLI[models.Flags]{}
	logger log.Interface
	db     *ent.Client
	cfg    *models.Config
	app    *gui.Gui
)

func exit() {
	if db != nil {
		db.Close()
	}

	if app != nil && app.IsRunning() {
		app.Stop()
	}
}

func main() {
	ctx := setup()

	logger.Infof("Starting HTTP server on %s:%d", cfg.HTTP.ListeningAddr, cfg.HTTP.Port)
	if err := chix.RunContext(ctx, httpServer(ctx)); err != nil {
		exit()
		if err.Error() != "received signal:" && cli.Flags.Mode == "cli" {
			if app == nil {
				app = gui.New(cfg, false)
			}

			app.Start(true, err)
			utils.SetConfig(ctx, cfg)
			fmt.Println("Re-configuration is complete, please try restarting Local GPSS now!")
		}
		return
	}
	exit()
}
