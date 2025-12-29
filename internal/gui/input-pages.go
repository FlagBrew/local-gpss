package gui

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"entgo.io/ent/dialect"
	"github.com/FlagBrew/local-gpss/internal/models"
	"github.com/gdamore/tcell/v2"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/rivo/tview"
	_ "modernc.org/sqlite"
)

var blackListedChars = []rune{
	'\'', '$', '%', '@', '#', '!', ';', ':', '/', '*', '?', '|', '>', '<', '&', '\\',
}

func (g *Gui) databaseConfigPage(p *tview.Pages, database string, allowExistingSqlite, skipImport bool) tview.Primitive {
	form := tview.NewForm()

	var values []string
	var fieldNames []string
	connectionString := g.config.Database.ConnectionString

	//changed := false
	switch database {
	case "sqlite":
		values = make([]string, 1)
		fieldNames = []string{"File Name"}
		if connectionString != "" && strings.HasPrefix(connectionString, "file:") {
			val := strings.TrimPrefix(connectionString, "file:")
			vals := strings.Split(val, "?")
			if len(vals) > 1 {
				values[0] = vals[0]
			} else {
				values[0] = "local-gpss.db"
			}
		} else {
			values[0] = "local-gpss.db"
		}
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

		if connectionString != "" {
			if database == "postgres" && strings.HasPrefix(connectionString, "postgres:") {
				conConf, err := pgx.ParseConfig(connectionString)
				if err == nil {
					values[0] = conConf.User
					values[1] = conConf.Password
					values[2] = conConf.Host
					values[3] = strconv.FormatUint(uint64(conConf.Port), 10)
					values[4] = conConf.Database
				}
			} else if strings.Contains(connectionString, "@tcp(") {
				conConf, err := mysql.ParseDSN(connectionString)
				if err == nil {
					values[0] = conConf.User
					values[1] = conConf.Passwd
					addrSplit := strings.Split(conConf.Addr, ":")
					values[2] = addrSplit[0]
					values[3] = addrSplit[1]
					values[4] = conConf.DBName
				}
			}
		}

		form.AddInputField(fieldNames[0], values[0], 20, nil, func(text string) {
			values[0] = text
		})
		form.AddPasswordField(fieldNames[1], values[1], 20, '*', func(text string) {
			values[1] = text
		})
		form.AddInputField(fieldNames[2], values[2], 20, nil, func(text string) {
			values[2] = text
		})
		form.AddInputField(fieldNames[3], values[3], 20, func(textToCheck string, lastChar rune) bool {
			if !unicode.IsDigit(lastChar) {
				return false
			}

			// Make sure the port is between 1 and 65535
			num, _ := strconv.Atoi(textToCheck)

			return num > 0 && num <= 65535
		}, func(text string) {
			values[3] = text
		})
		form.AddInputField(fieldNames[4], values[4], 20, nil, func(text string) {
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
				connectionString = fmt.Sprintf("file:%s?cache=shared&_pragma=foreign_keys(1)", path)
				break
			} else if !allowExistingSqlite {
				errors = append(errors, "File Name: this file already exists")
			}

			// Try connecting
			db, err := sql.Open("sqlite", connectionString)
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
			g.config.Database = models.DatabaseConfig{
				DBType:           database,
				ConnectionString: connectionString,
			}

			if skipImport {
				p.AddPage("http-config", g.httpConfigPage(p), true, false)
				p.SwitchToPage("http-config")
			} else {
				p.AddPage("database-import", g.databaseDownload(p), true, false)
				p.SwitchToPage("database-import")
			}

		}

	})

	return frame
}

func (g *Gui) httpConfigPage(p *tview.Pages) tview.Primitive {
	form := tview.NewForm()
	frame := tview.NewFrame(form)

	chosenAddr := "0.0.0.0"
	chosenPort := "8080"

	if g.config.HTTP.ListeningAddr != "" {
		chosenAddr = g.config.HTTP.ListeningAddr
	}

	if g.config.HTTP.Port != 0 {
		chosenPort = fmt.Sprintf("%d", g.config.HTTP.Port)
	}

	defaultFrameDraw := func() {
		frame.Clear()
		frame.AddText("Please fill out the form below, with the information (if you are unsure, ask for assistance on Discord)", true, tview.AlignLeft, tcell.ColorYellow)
		frame.AddText("[red]ESC - exit[-:-:-:-] [yellow] Enter - next input/submit [orange] (Shift+)Tab - switch inputs", false, tview.AlignLeft, tcell.ColorYellow)
		if chosenAddr == "127.0.0.1" || chosenAddr == "localhost" || chosenAddr == "::1" {
			frame.AddText(fmt.Sprintf("Using %s (localhost) is only recommended if running Docker", chosenAddr), true, tview.AlignLeft, tcell.ColorRed)
		}
	}

	defaultFrameDraw()
	availableAddresses := []string{"0.0.0.0"}

	ipHelpText := `
When selecting the listening address, 0.0.0.0 will have Local GPSS listen on all IP addresses bound to your computer
For most users, that is perfectly fine, as your system's IP address is likely dynamic and will change after a period of time.

For advance users, you may specifically want to bind to a specific IP address and you may choose it from the dropdown list.
Do keep in mind, that if the IP assignment changes, you will need to update the configuration file.
`

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		ipHelpText = fmt.Sprintf(`Due to an error, the application couldn't list the IPs assigned to your machine, as such it will fallback to using 0.0.0.0 (listening on all interfaces)
Error info: %s
`, err.Error())
	} else {
		for _, address := range addrs {
			if strings.HasPrefix(address.String(), "fe80") {
				continue
			}
			availableAddresses = append(availableAddresses, strings.Split(address.String(), "/")[0])
		}
	}

	index := slices.Index(availableAddresses, chosenAddr)
	if index == -1 {
		index = 0
	}

	form.AddTextView("IP Info", ipHelpText, 0, 0, true, true)
	form.AddDropDown("Listening Address", availableAddresses, index, func(option string, optionIndex int) {
		chosenAddr = availableAddresses[optionIndex]
		defaultFrameDraw()
	})
	form.AddTextView("Port Info", `When choosing the port, keep in mind the following: 
1. the port must be between 1 and 65535 
2. on certain platforms (such as Linux), ports 1-1023 are "privileged ports", meaning you need to be running the server as root [::b](STRONGLY NOT RECOMMENDED)[-:-:-:-] to bind to them.
The default port (8080) should be good for most users, if it's in use, try incrementing it.`, 0, 0, true, true)
	form.AddInputField("Port", chosenPort, 20, func(textToCheck string, lastChar rune) bool {
		if !unicode.IsDigit(lastChar) {
			return false
		}

		// Make sure the port is between 1 and 65535
		num, _ := strconv.Atoi(textToCheck)

		return num > 0 && num <= 65535
	}, func(text string) {
		chosenPort = text
	})

	form.AddButton("Submit", func() {
		defaultFrameDraw()
		errors := []string{}
		if chosenPort == "" {
			errors = append(errors, "Port: Please enter a valid port number")
		}

		if len(errors) > 0 {
			frame.AddText("Errors: ", true, tview.AlignLeft, tcell.ColorYellow)
			for _, v := range errors {
				frame.AddText(v, true, tview.AlignLeft, tcell.ColorRed)
			}
		} else {
			addr := chosenAddr + ":" + chosenPort
			if strings.Contains(chosenAddr, ":") {
				addr = "[" + chosenAddr + "]:" + chosenPort
			}
			l, err := net.Listen("tcp", addr)

			if err != nil {
				frame.AddText("Errors: ", true, tview.AlignLeft, tcell.ColorYellow)
				frame.AddText(err.Error(), true, tview.AlignLeft, tcell.ColorRed)
				return
			}

			l.Close()

			port, _ := strconv.Atoi(chosenPort)

			// Update the config with the new settings
			go g.app.QueueUpdateDraw(func() {
				g.config.HTTP = models.HTTPConfig{
					ListeningAddr: chosenAddr,
					Port:          port,
				}
				//g.app.Draw()
			})

			p.SwitchToPage("display-config")
		}
	})

	frame.SetBorder(true)
	frame.SetTitle("Local GPSS - Configuring HTTP")

	return frame
}
