package gui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (g *Gui) introPage(p *tview.Pages) tview.Primitive {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true)

	textView.SetText(`Welcome to Local GPSS, if this is the first time you're running local GPSS, [::b]I strongly recommend you read the wiki[-:-:-:-] (https://github.com/FlagBrew/local-gpss/wiki)

This wizard will walk you through setting up your configuration, and if you have any problems, please let us know on Discord!

If you would like to exit the wizard early, please press the [red]esc key[-:-:-:-], otherwise please press [yellow]enter[-:-:-:-] to continue

`)

	textView.SetBorder(true).SetTitle("Local GPSS")
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			p.SwitchToPage("database-type")
		}
		return event
	})

	frame := tview.NewFrame(textView)
	frame.AddText("[red]ESC - exit[-:-:-:-] [yellow] Enter - continue", false, tview.AlignLeft, tcell.ColorYellow)

	return frame
}
