package gui

import (
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (g *Gui) databaseSelection(p *tview.Pages) tview.Primitive {
	list := tview.NewList()

	list.AddItem("sqlite", "Easiest to use, creates a database file on disk, good if running for yourself only, [::b]if you have no experience with databases, use this option", '1', func() {
		p.AddPage("db-config", g.databaseConfigPage(p, "sqlite"), true, false)
		p.SwitchToPage("db-config")
	})
	list.AddItem("MySql", "Requires a running instance of a MySql database recommended if sharing Local GPSS instance with others", '2', func() {
		p.AddPage("db-config", g.databaseConfigPage(p, "mysql"), true, false)
		p.SwitchToPage("db-config")
	})
	list.AddItem("Postgres", "Requires a running instance of a Postgres database, recommended if sharing Local GPSS instance with others. Alternative (and arguably better than) to MySql", '3', func() {
		p.AddPage("db-config", g.databaseConfigPage(p, "postgres"), true, false)
		p.SwitchToPage("db-config")
	})

	frame := tview.NewFrame(list)
	frame.SetBorder(true)
	frame.SetTitle("Local GPSS - Choosing Database")
	frame.AddText("Please select below what database you would like to use for Local GPSS", true, tview.AlignLeft, tcell.ColorYellow)
	frame.AddText("[red]ESC - exit[-:-:-:-] [yellow] Enter - continue", false, tview.AlignLeft, tcell.ColorYellow)
	return frame
}

func (g *Gui) databaseDownload(p *tview.Pages) tview.Primitive {
	list := tview.NewList()
	frame := tview.NewFrame(list)
	frame.SetBorder(true)
	frame.AddText("[red]ESC - exit[-:-:-:-] [yellow] Enter - continue", false, tview.AlignLeft, tcell.ColorYellow)
	frame.SetTitle("Local GPSS - Import Database")

	// Check if there's a gpss.db that already exists
	if _, err := os.Stat("gpss.db"); os.IsNotExist(err) {

		frame.AddText("Local GPSS has detected you do not have an already existing database, would you like to download the archived database?", true, tview.AlignLeft, tcell.ColorYellow)
		list.AddItem("Yes and Re-run legality checks", "This will download the database from GitHub and have the legality checks re-done (this will take some time)", '1', func() {
			g.config.Misc.DownloadOriginalDb = true
			g.config.Misc.RecheckLegality = true
			g.config.Misc.MigrateOriginalDb = true
			p.AddPage("http-config", g.httpConfigPage(p), true, false)
			p.SwitchToPage("http-config")
		})
		list.AddItem("Yes", "This will download the database from GitHub but will not re-run legality checks", '2', func() {
			g.config.Misc.DownloadOriginalDb = true
			g.config.Misc.RecheckLegality = false
			g.config.Misc.MigrateOriginalDb = true
			p.AddPage("http-config", g.httpConfigPage(p), true, false)
			p.SwitchToPage("http-config")
		})
		list.AddItem("No", "You will start with a fresh empty database", '3', func() {
			g.config.Misc.DownloadOriginalDb = false
			g.config.Misc.RecheckLegality = false
			g.config.Misc.MigrateOriginalDb = false
			p.AddPage("http-config", g.httpConfigPage(p), true, false)
			p.SwitchToPage("http-config")
		})
	} else if err == nil {
		frame.AddText("Local GPSS has detected you do have an already existing database, Would you like to migrate it?", true, tview.AlignLeft, tcell.ColorYellow)
		list.AddItem("Yes and Re-run legality checks", "This will migrate the old database and have the legality checks re-done (this will take some time)", '1', func() {
			g.config.Misc.DownloadOriginalDb = false
			g.config.Misc.RecheckLegality = true
			g.config.Misc.MigrateOriginalDb = true
			p.AddPage("http-config", g.httpConfigPage(p), true, false)
			p.SwitchToPage("http-config")
		})
		list.AddItem("Yes", "This will migrate the old database but will not re-run legality checks", '2', func() {
			g.config.Misc.DownloadOriginalDb = false
			g.config.Misc.RecheckLegality = false
			g.config.Misc.MigrateOriginalDb = true
			p.AddPage("http-config", g.httpConfigPage(p), true, false)
			p.SwitchToPage("http-config")
		})
		list.AddItem("No", "You will start with a fresh empty database", '3', func() {
			g.config.Misc.DownloadOriginalDb = false
			g.config.Misc.RecheckLegality = false
			g.config.Misc.MigrateOriginalDb = false
			p.AddPage("http-config", g.httpConfigPage(p), true, false)
			p.SwitchToPage("http-config")
		})
	}

	return frame
}

func (g *Gui) displayMode(p *tview.Pages) tview.Primitive {
	list := tview.NewList()

	list.AddItem("simple", "Plain and simple, no fancy terminal GUI, just plain-ol logs.", '1', func() {
		p.AddPage("confirm", g.confirmationPage(p), true, false)
		g.config.FancyScreen = false
		p.SwitchToPage("confirm")
	})
	list.AddItem("fancy", "Displays the application in a fancy layout similar to that of the set-up wizard, will allow for updating configuration file within the app.", '2', func() {
		p.AddPage("confirm", g.confirmationPage(p), true, false)
		g.config.FancyScreen = true
		p.SwitchToPage("confirm")
	})

	frame := tview.NewFrame(list)
	frame.SetBorder(true)
	frame.SetTitle("Local GPSS - Choosing Display Mode")
	frame.AddText("Please select below which display mode you would like to use when running Local GPSS", true, tview.AlignLeft, tcell.ColorYellow)
	frame.AddText("[red]ESC - exit[-:-:-:-] [yellow] Enter - continue", false, tview.AlignLeft, tcell.ColorYellow)
	return frame
}
