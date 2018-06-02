package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"github.com/urfave/cli"
	"golang.org/x/oauth2"
)

const version = "0.1.2"

type subCmd interface {
	Run(*cli.Context, *config, *github.Client) error
}

var cmdList = []cli.Command{}

func main() {

	app := cli.NewApp()
	app.Name = "gitHubManager"
	app.Version = version
	app.Usage = "This tool helps you to manage project with GitHub."

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

	app.Commands = cmdList

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func action(c *cli.Context, sc subCmd) error {

	conf, err := readConfig(c)
	if err != nil {
		return err
	}

	ctx := context.Background()
	var client *http.Client
	if conf.Token != nil {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *conf.Token},
		)
		client = oauth2.NewClient(ctx, ts)
	} else {
		client = http.DefaultClient
	}

	return sc.Run(c, conf, github.NewClient(client))
}
