package gui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (g *Gui) databaseSelection(p *tview.Pages) tview.Primitive {
	list := tview.NewList()
	list.AddItem("sqlite", "Easiest to use, creates a database file on disk, good if running for yourself only, [::b]if you have no experience with databases, use this option", '1', func() {
		p.SwitchToPage("sqlite")
	})
	list.AddItem("MySql", "Requires a running instance of a MySql database recommended if sharing Local GPSS instance with others", '2', func() {
		p.SwitchToPage("mysql")
	})
	list.AddItem("Postgres", "Requires a running instance of a Postgres database, recommended if sharing Local GPSS instance with others. Alternative (and arguably better than) to MySql", '3', func() {
		p.SwitchToPage("postgres")
	})

	frame := tview.NewFrame(list)
	frame.SetBorder(true)
	frame.SetTitle("Local GPSS - Choosing Database")
	frame.AddText("Please select below what database you would like to use for Local GPSS", true, tview.AlignLeft, tcell.ColorYellow)
	frame.AddText("[red]ESC - exit[-:-:-:-] [yellow] Enter - continue", false, tview.AlignLeft, tcell.ColorYellow)
	return frame
}
