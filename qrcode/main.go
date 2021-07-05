package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"qrcode"
)

func main() {
	var url string
	var err error
	var q *qrcode.QRCode

	app := &cli.App{
		Name:  "Qrcode",
		Usage: "Qrcode is a tool for print qrcode in terminal",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Load configuiration file",
				Destination: &url,
			},
		},
		Action: func(c *cli.Context) error {
			path := ""
			if c.NArg() > 0 {
				path = c.Args().Get(0)
				log.Println(path)

				q, err = qrcode.New(path, qrcode.Highest)
				checkError(err)

				art := q.ToString(false)
				fmt.Println(art)

				// run qrcode
				return nil
			}
			log.Fatal(url)
			return nil
		},
	}

	//
	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	// customization cli help template
	cli.AppHelpTemplate = fmt.Sprintf(`%s
  SUPPORT: git
  `, cli.AppHelpTemplate)

	cli.AppHelpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
COMMANDS:
{{range .Commands}}{{if not .HideHelp}}   {{join .Names ", "}}{{ "\t"}}{{.Usage}}{{ "\n" }}{{end}}{{end}}{{end}}{{if .VisibleFlags}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}{{end}}{{if .Copyright }}
COPYRIGHT:
   {{.Copyright}}
   {{end}}{{if .Version}}
VERSION:
   {{.Version}}
   {{end}}
`
}

func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
