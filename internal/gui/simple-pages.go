package gui

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/rivo/tview"
)

func (g *Gui) introPage(p *tview.Pages) tview.Primitive {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	textView.SetText(`Welcome to Local GPSS, if this is the first time you're running local GPSS, [::b]I strongly recommend you read the wiki[-:-:-:-] (https://github.com/FlagBrew/local-gpss/wiki)

This wizard will walk you through setting up your configuration, and if you have any problems, please let us know on Discord!

[::b]It is strongly recommended that you maximize this terminal window to avoid text being cut-off[-:-:-:-]

If you would like to exit the wizard early, please press the [red]esc key[-:-:-:-], otherwise please press [yellow]enter[-:-:-:-] to continue

`)

	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			p.SwitchToPage("database-type")
		}
		return event
	})

	frame := tview.NewFrame(textView)
	frame.AddText("[red]ESC - exit[-:-:-:-] [yellow] Enter - continue", false, tview.AlignLeft, tcell.ColorYellow)
	frame.SetBorder(true).SetTitle("Local GPSS")
	return frame
}

func (g *Gui) confirmationPage(p *tview.Pages) tview.Primitive {
	form := tview.NewForm()

	databaseText := ""
	displayMode := "simple"
	if g.config.FancyScreen {
		displayMode = "fancy"
	}

	switch g.config.Database.DBType {
	case "sqlite":
		uri, err := url.Parse(g.config.Database.ConnectionString)
		if err != nil {
			databaseText = "Failed to parse database connection string"
		} else {
			databaseText = fmt.Sprintf(`Type: Sqlite
File: %s`, uri.Host)
		}
	case "postgres":
		conConf, err := pgx.ParseConfig(g.config.Database.ConnectionString)
		if err != nil {
			databaseText = "Failed to parse database connection string"
		} else {
			databaseText = fmt.Sprintf(`Type: Postgres
User: %s, Password: %s
Host: %s, Port: %d
DB Name: %s
`, conConf.User, strings.Repeat("*", len(conConf.Password)), conConf.Host, conConf.Port, conConf.Database)
		}
	case "mysql":
		conConf, err := mysql.ParseDSN(g.config.Database.ConnectionString)
		if err != nil {
			databaseText = "Failed to parse database connection string"
		} else {
			addrSplit := strings.Split(conConf.Addr, ":")
			databaseText = fmt.Sprintf(`Type: Mysql
User: %s, Password: %s
Host: %s, Port: %s
DB Name: %s
`, conConf.User, strings.Repeat("*", len(conConf.Passwd)), addrSplit[0], addrSplit[1], conConf.DBName)
		}
	}

	form.AddTextView("Database Settings", databaseText, 0, 0, true, true)
	form.AddTextView("HTTP Settings", fmt.Sprintf(`Listening Address: %s
Listening Port: %d
`, g.config.HTTP.ListeningAddr, g.config.HTTP.Port), 0, 0, true, true)
	form.AddTextView("Display Mode", displayMode, 0, 0, true, true)
	form.AddTextView("Import Options", fmt.Sprintf(`Download GPSS Database: %t
Import Data: %t
Rerun Legality Check: %t
`, g.config.Misc.DownloadOriginalDb, g.config.Misc.MigrateOriginalDb, g.config.Misc.RecheckLegality), 0, 0, true, true)

	form.AddButton("Save", func() {
		g.Stop()
	})
	form.AddButton("Edit", func() {
		p.SwitchToPage("database-type")
	})

	frame := tview.NewFrame(form)
	frame.AddText("Please review the details below, and if all is good, press enter on the save button, otherwise, press the edit button to go back to the first page (with your data saved of course)", true, tview.AlignLeft, tcell.ColorYellow)
	frame.AddText("[red]ESC - exit[-:-:-:-] [yellow] Enter - submit [orange] (Shift+)Tab - switch buttons", false, tview.AlignLeft, tcell.ColorYellow)
	frame.SetBorder(true)
	frame.SetTitle("Local GPSS - Settings Review")

	return frame
}

func (g *Gui) mainPage(p *tview.Pages) tview.Primitive {

	textView := tview.NewTextView()
	frame := tview.NewFrame(textView)
	followLogs := true
	followLogsText := "[green]f/F - Follow Logs: On[-:-:-:-]"
	monCount := -1
	bundleCount := -1
	firstQuery := true

	redrawFrame := func() {
		frame.Clear()
		frame.AddText(fmt.Sprintf("[red]ESC/Q/q - exit[-:-:-:-] | [red]c/C - Clear Logs [-:-:-:-] | %s", followLogsText), false, tview.AlignLeft, tcell.ColorYellow)
		frame.AddText(fmt.Sprintf("Listening on %s:%d", g.config.HTTP.ListeningAddr, g.config.HTTP.Port), false, tview.AlignLeft, tcell.ColorYellow)
		if monCount != -1 && bundleCount != -1 {
			frame.AddText(fmt.Sprintf("Pokemon in DB: %d, Bundles in DB: %d", monCount, bundleCount), false, tview.AlignRight, tcell.ColorYellow)
		} else {
			frame.AddText("DB Stats Loading, Please wait...", false, tview.AlignRight, tcell.ColorYellow)
		}

	}

	frame.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'f', 'F':
				followLogs = !followLogs
				if followLogs {
					followLogsText = "[green]f/F - Follow Logs: On[-:-:-:-]"
				} else {
					followLogsText = "[red]f/F - Follow Logs: Off[-:-:-:-]"
				}

				redrawFrame()
			case 'c', 'C':
				textView.Clear()
			case 'q', 'Q':
				g.Stop()
			}

		}
		return event
	})

	go func() {
		for {
			if firstQuery {
				time.Sleep(500 * time.Millisecond)
			} else {
				time.Sleep(5 * time.Second)
			}

			if g.db == nil {
				continue
			}

			mons, err := g.db.Pokemon.Query().Count(context.Background())
			if err == nil {
				monCount = mons
			}

			bundles, err := g.db.Bundle.Query().Count(context.Background())
			if err == nil {
				bundleCount = bundles
			}

			firstQuery = false
			redrawFrame()
			if !g.running {
				break
			}
		}

	}()

	go func() {
		for {
			g.app.QueueUpdateDraw(func() {
				if followLogs {
					textView.ScrollToEnd()
				}
			})
			time.Sleep(time.Millisecond * 500)
			if !g.running {
				break
			}
		}
	}()

	g.logOutput = textView

	textView.SetMaxLines(100)

	redrawFrame()

	frame.SetBorder(true)
	frame.SetTitle("Local GPSS")
	return frame
}
