package gui

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"
	"unicode"

	"entgo.io/ent/dialect"
	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/gdamore/tcell/v2"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/rivo/tview"
	_ "modernc.org/sqlite"
)

var blackListedChars = []rune{
	'\'', '$', '%', '@', '#', '!', ';', ':', '/', '*', '?', '|', '>', '<', '&', '\\',
}

func (g *Gui) databaseConfigPage(p *tview.Pages, database string) tview.Primitive {
	form := tview.NewForm()

	var values []string
	var fieldNames []string
	connectionString := ""

	//changed := false
	switch database {
	case "sqlite":
		values = make([]string, 1)
		fieldNames = []string{"File Name"}
		values[0] = "local-gpss.db"
		form.AddInputField(fieldNames[0], values[0], 20, func(textToCheck string, lastChar rune) bool {
			if slices.Contains(blackListedChars, lastChar) {
				return false
			}
			return true
		}, func(text string) {
			values[0] = text
		})
	case "mysql", "postgres":
		values = make([]string, 5)
		fieldNames = []string{"Username", "Password", "Host", "Port", "Database"}
		form.AddInputField(fieldNames[0], "", 20, nil, func(text string) {
			values[0] = text
		})
		form.AddPasswordField(fieldNames[1], "", 20, '*', func(text string) {
			values[1] = text
		})
		form.AddInputField(fieldNames[2], "", 20, nil, func(text string) {
			values[2] = text
		})
		form.AddInputField(fieldNames[3], "", 20, func(textToCheck string, lastChar rune) bool {
			return unicode.IsDigit(lastChar)
		}, func(text string) {
			values[3] = text
		})
		form.AddInputField(fieldNames[4], "", 20, nil, func(text string) {
			values[4] = text
		})

	}

	frame := tview.NewFrame(form)
	frame.SetBorder(true)
	frame.SetTitle(fmt.Sprintf("Local GPSS - Configuring Database: %s", database))
	frame.AddText("Please fill out the form below, with the information (if you are unsure, ask for assistance on Discord)", true, tview.AlignLeft, tcell.ColorYellow)
	frame.AddText("[red]ESC - exit[-:-:-:-] [yellow] Enter - next input/submit [orange] (Shift+)Tab - switch inputs", false, tview.AlignLeft, tcell.ColorYellow)
	form.AddButton("Submit", func() {
		frame.Clear()
		frame.AddText("Please fill out the form below, with the information (if you are unsure, ask for assistance on Discord)", true, tview.AlignLeft, tcell.ColorYellow)
		frame.AddText("[red]ESC - exit[-:-:-:-] [yellow] Enter - next input/submit [orange] (Shift+)Tab - switch inputs", false, tview.AlignLeft, tcell.ColorYellow)
		errors := []string{}
		// Basic validation first

		for i, fieldName := range fieldNames {
			if values[i] == "" {
				errors = append(errors, fmt.Sprintf("%s: is required", fieldName))
			}
		}

		switch database {
		case "sqlite":
			if values[0] == "" {
				break
			}

			path := filepath.Join(values[0])
			if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
				errors = append(errors, "File Name: an unknown error occurred, please check your input")
			} else if os.IsNotExist(err) {
				connectionString = fmt.Sprintf("file://%s?cache=shared&_fk=1", path)
				break
			} else {
				errors = append(errors, "File Name: this file already exists")
			}

			// Try connecting
			db, err := sql.Open(dialect.SQLite, connectionString)
			if err != nil {
				errors = append(errors, "Sqlite connection error: "+err.Error())
				break
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			err = db.PingContext(ctx)
			if err != nil {
				errors = append(errors, "Sqlite connection error: "+err.Error())
			}
			db.Close()
		case "mysql", "postgres":
			if values[0] == "" || values[1] == "" || values[2] == "" || values[3] == "" || values[4] == "" {
				break
			}

			port, err := strconv.Atoi(values[3])
			if err != nil {
				errors = append(errors, "Port: input is invalid")
			} else {
				if port < 1 || port > 65535 {
					errors = append(errors, "Port: input is out of range (1 - 65535)")
				}
			}

			// Try connecting to make sure it is valid
			if database == "postgres" {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				connectionString = fmt.Sprintf("postgres://%s:%s@%s:%d/%s", values[0], url.QueryEscape(values[1]), values[2], port, values[4])
				db, err := pgx.Connect(ctx, connectionString)
				if err != nil {
					errors = append(errors, "Postgres connection error: "+err.Error())
					break
				}

				err = db.Ping(ctx)
				if err != nil {
					errors = append(errors, "Postgres connection error: "+err.Error())
				}
				db.Close(context.Background())
			} else {
				connectionString = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", values[1], values[2], values[3], port, values[4])
				db, err := sql.Open(dialect.MySQL, connectionString)
				if err != nil {
					errors = append(errors, "MySQL connection error: "+err.Error())
					break
				}

				ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
				defer cancel()
				err = db.PingContext(ctx)
				if err != nil {
					errors = append(errors, "MySQL connection error: "+err.Error())
				}
				db.Close()
			}

		}

		if len(errors) > 0 {
			frame.AddText("Errors: ", true, tview.AlignLeft, tcell.ColorYellow)
			for _, v := range errors {
				frame.AddText(v, true, tview.AlignLeft, tcell.ColorRed)
			}
		} else {
			// Update the config
			g.createdConfig.Database = models.DatabaseConfig{
				DBType:           database,
				ConnectionString: connectionString,
			}

			p.SwitchToPage("http-config")

		}

	})

	return frame
}
