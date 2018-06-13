package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/github"
	"github.com/urfave/cli"
)

func init() {
	cmdList = append(cmdList, cli.Command{
		Name:  "label",
		Usage: "check or set up label settings with json file",
		Action: func(c *cli.Context) error {
			return action(c, &label{Out: os.Stdout})
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "file, f",
				Value: "label_settings.json",
				Usage: "you can change label setting json file",
			},
			cli.BoolFlag{
				Name:  "update, u",
				Usage: "if this option is set, send update request to github",
			},
		},
	})
}

type label struct {
	Out io.Writer
}

func (l label) Run(c *cli.Context, conf *config, client *github.Client) error {

	setting, err := l.ReadSettings(c.String("file"))
	if err != nil {
		return err
	}

	labels, _, err := client.Issues.ListLabels(context.Background(), *conf.User, *conf.Repo, nil)
	if err != nil {
		return err
	}

	currentLabelMap := make(map[string]*struct {
		Color     string
		IsDefined bool
	})
	for _, v := range labels {
		currentLabelMap[*v.Name] = &struct {
			Color     string
			IsDefined bool
		}{*v.Color, false}
	}

	// TODO select check or update

	// output
	fmt.Fprintf(l.Out, "# Label settings for `%s/%s`\n", *conf.User, *conf.Repo)

	if len(setting.Labels) > 0 {
		fmt.Fprintln(l.Out, "  * label settings")
		for _, v := range setting.Labels {
			if cl, ok := currentLabelMap[v.Name]; ok {
				cl.IsDefined = true
				fmt.Fprintf(l.Out, "    `%s`: update (color=\"%s\" -> \"%s\", desc=\"%s\")\n", v.Name, cl.Color, v.Color, v.Desc)
			} else {
				fmt.Fprintf(l.Out, "    `%s`: create (color=\"%s\", desc=\"%s\")\n", v.Name, v.Color, v.Desc)
			}
		}
		fmt.Fprintln(l.Out, "")
	}

	if len(setting.Replace) > 0 {
		fmt.Fprintln(l.Out, "  * replace labels")
		for _, v := range setting.Replace {
			if cl, ok := currentLabelMap[v.From]; ok {
				cl.IsDefined = true
				// TODO check issue
				fmt.Fprintf(l.Out, "    `%s`: replace to `%s`\n", v.From, v.To)
			} else {
				fmt.Fprintf(l.Out, "    `%s`: don't exist in this repository\n", v.From)
			}
		}
		fmt.Fprintln(l.Out, "")
	}

	if len(setting.Ignore) > 0 {
		existIgnore := make(map[string]int)
		fmt.Fprintln(l.Out, "  * ignore labels")
		for k, v := range currentLabelMap {
			if v.IsDefined {
				continue
			}
			for _, v2 := range setting.Ignore {
				ptn := regexp.MustCompile("^" + v2 + "$")
				if ptn.Match([]byte(k)) {
					v.IsDefined = true
					existIgnore[v2] = 1
					fmt.Fprintf(l.Out, "    `%s`\n", k)
				}
			}
		}
		for _, v := range setting.Ignore {
			if _, ok := existIgnore[v]; ok {
				continue
			}
			fmt.Fprintf(l.Out, "    `%s`: don't exist in this repository\n", v)
		}
		fmt.Fprintln(l.Out, "")
	}

	fmt.Fprintln(l.Out, "  * delete labels")
	for k, v := range currentLabelMap {
		if v.IsDefined {
			continue
		}
		// TODO check issue
		fmt.Fprintf(l.Out, "    `%s`\n", k)
	}
	fmt.Fprintln(l.Out, "")

	return nil
}

func (l label) ReadSettings(filename string) (*labelSetting, error) {

	setting := &labelSetting{}

	jsonStr, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("not found config file (%s)", filename)
	}

	err = json.Unmarshal(jsonStr, setting)
	if err != nil {
		return nil, fmt.Errorf("something wrong in setting file (%s)", filename)
	}

	// create LabelMap and check
	setting.LabelMap = make(map[string]labelItem)
	for _, v := range setting.Labels {
		if _, ok := setting.LabelMap[v.Name]; ok {
			return nil, fmt.Errorf("label name is duplicated in setting file (%s)", v.Name)
		}
		setting.LabelMap[v.Name] = labelItem{Color: v.Color, Desc: v.Desc}
	}

	for _, v := range setting.Replace {
		if _, ok := setting.LabelMap[v.From]; ok {
			return nil, fmt.Errorf("label name is duplicated in setting file (%s)", v.From)
		}
		if _, ok := setting.LabelMap[v.To]; !ok {
			return nil, fmt.Errorf("label name of `replace - to` is not found in labels (%s)", v.To)
		}
		setting.LabelMap[v.From] = labelItem{ReplaceTo: v.To}
	}

	for i, v := range setting.Ignore {
		v2 := strings.Replace(v, "*", ".*", -1)
		if v != v2 {
			setting.Ignore[i] = v2
		}
		if _, ok := setting.LabelMap[v2]; ok {
			return nil, fmt.Errorf("label name is duplicated in setting file (%s)", v)
		}
		setting.LabelMap[v2] = labelItem{IsIgnore: true}
	}

	return setting, nil
}

type labelSetting struct {
	Labels []struct {
		Name  string `json:"name"`
		Color string `json:"color"`
		Desc  string `json:"desc"`
	} `json:"labels"`
	Replace []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"replace"`
	Ignore   []string `json:"ignore"`
	LabelMap map[string]labelItem
}

type labelItem struct {
	Color, Desc, ReplaceTo string
	IsIgnore               bool
}
