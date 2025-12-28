package gui

import (
	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Gui struct {
	app           *tview.Application
	createdConfig models.Config
}

func New() *Gui {
	app := &Gui{
		app:           tview.NewApplication(),
		createdConfig: models.Config{},
	}

	app.app.EnableMouse(true)
	app.Init()

	return app
}

func (g *Gui) Init() {
	pages := tview.NewPages()
	pages.AddPage("main", g.introPage(pages), true, true)
	// Database Pages
	pages.AddPage("database-type", g.databaseSelection(pages), true, false)
	pages.AddPage("sqlite", g.databaseConfigPage(pages, "sqlite"), true, false)
	pages.AddPage("mysql", g.databaseConfigPage(pages, "mysql"), true, false)
	pages.AddPage("postgres", g.databaseConfigPage(pages, "postgres"), true, false)

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			g.app.Stop()
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
