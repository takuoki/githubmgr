package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

const (
	version = "0.1.1"
)

func main() {

	app := cli.NewApp()
	app.Name = "gitHubManager"
	app.Version = version
	app.Usage = "This tool helps you to manage project with GitHub."

	app.Commands = []cli.Command{
		issueCmd,
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Value: "config.json",
			Usage: "configure file for this tool",
		},
		cli.StringFlag{
			Name:  "user, u",
			Value: "",
			Usage: "username for GitHub",
		},
		cli.StringFlag{
			Name:  "repo, r",
			Value: "",
			Usage: "repository name for GitHub",
		},
		cli.StringFlag{
			Name:  "token, t",
			Value: "",
			Usage: "access token to connect your ripository",
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err.Error())
		return
	}
}

func action(c *cli.Context, f func(*cli.Context, *config) error) error {

	conf, err := readConfig(c)
	if err != nil {
		return err
	}

	return f(c, conf)
}
