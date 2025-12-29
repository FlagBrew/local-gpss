package gui

import (
	"os"

	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Gui struct {
	app    *tview.Application
	config *models.Config
}

func New(config *models.Config) *Gui {
	app := &Gui{
		app:    tview.NewApplication(),
		config: &models.Config{},
	}

	if config != nil {
		app.config = config
	}

	app.app.EnableMouse(true)

	app.Init()

	return app
}

func (g *Gui) Init() {
	pages := tview.NewPages()
	pages.AddPage("setup", g.introPage(pages), true, true)
	pages.AddPage("database-type", g.databaseSelection(pages), true, false)

	// Other settings
	pages.AddPage("display-config", g.displayMode(pages), true, false)

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			g.app.Stop()
			os.Exit(0)
		}
		return event
	})

	g.app.SetRoot(pages, true)
}

func (g *Gui) Start() error {
	err := g.app.Run()
	if err != nil {
		return err
	}

	return nil
}

func (g *Gui) Stop() {
	g.app.Stop()
}
