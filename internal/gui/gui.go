package gui

import (
	"io"
	"os"

	"github.com/FlagBrew/local-gpss/internal/database/ent"
	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Gui struct {
	app       *tview.Application
	config    *models.Config
	running   bool
	logOutput io.Writer
	db        *ent.Client
}

func New(config *models.Config, wizard bool) *Gui {
	app := &Gui{
		app:       tview.NewApplication(),
		config:    &models.Config{},
		running:   false,
		logOutput: os.Stdout,
	}

	if config != nil {
		app.config = config
	}

	app.app.EnableMouse(true)

	app.Init(wizard)

	return app
}

func (g *Gui) Init(wizard bool) {
	pages := tview.NewPages()

	if wizard {
		pages.AddPage("setup", g.introPage(pages), true, true)
		pages.AddPage("database-type", g.databaseSelection(pages), true, false)
		pages.AddPage("display-config", g.displayMode(pages), true, false)
	} else {
		pages.AddPage("main", g.mainPage(pages), true, true)
	}

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			g.app.Stop()
			if wizard {
				os.Exit(0)
			}

		}
		return event
	})

	g.app.SetRoot(pages, true)
}

func (g *Gui) Start() error {
	g.running = true
	err := g.app.Run()
	if err != nil {
		g.running = false
		return err
	}

	return nil
}

func (g *Gui) Stop() {
	g.running = false
	g.app.Stop()
}

func (g *Gui) GetLogOutput() io.Writer {
	return g.logOutput
}

func (g *Gui) SetDb(db *ent.Client) {
	g.db = db
}
