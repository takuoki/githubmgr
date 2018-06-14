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

	updOpes := []updOpe{}

	// output
	fmt.Fprintf(l.Out, "# Label settings for `%s/%s`\n", *conf.User, *conf.Repo)

	if len(setting.Labels) > 0 {
		fmt.Fprintln(l.Out, "  * label settings")
		for _, v := range setting.Labels {
			if cl, ok := currentLabelMap[v.Name]; ok {
				cl.IsDefined = true
				updOpes = append(updOpes, l.CreateUpdOpe(v.Name, opeUpd, v.Color, v.Desc, nil))
				fmt.Fprintf(l.Out, "    `%s`: update (color=\"%s\" -> \"%s\", desc=\"%s\")\n", v.Name, cl.Color, v.Color, v.Desc)
			} else {
				updOpes = append(updOpes, l.CreateUpdOpe(v.Name, opeCrt, v.Color, v.Desc, nil))
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
				issueNums, err := l.GetIssues(client, *conf.User, *conf.Repo, v.From)
				if err != nil {
					return err
				}
				if len(issueNums) > 0 {
					updOpes = append(updOpes, l.CreateUpdOpe(v.To, opeIss, "", "", issueNums))
					updOpes = append(updOpes, l.CreateUpdOpe(v.From, opeDel, "", "", nil))
					fmt.Fprintf(l.Out, "    `%s`: replace to `%s` (issues=%s) and delete\n", v.From, v.To, concatInt(issueNums, ", "))
				} else {
					updOpes = append(updOpes, l.CreateUpdOpe(v.From, opeDel, "", "", nil))
					fmt.Fprintf(l.Out, "    `%s`: delete (there are no issues attatched this label)\n", v.From)
				}
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
	existDelLabel := false
	existDelLabelWithIssue := false
	for k, v := range currentLabelMap {
		if v.IsDefined {
			continue
		}
		existDelLabel = true
		issueNums, err := l.GetIssues(client, *conf.User, *conf.Repo, k)
		if err != nil {
			return err
		}
		if len(issueNums) > 0 {
			existDelLabelWithIssue = true
			fmt.Fprintf(l.Out, "    `%s` (issues=%s)\n", k, concatInt(issueNums, ", "))
		} else {
			updOpes = append(updOpes, l.CreateUpdOpe(k, opeDel, "", "", nil))
			fmt.Fprintf(l.Out, "    `%s`\n", k)
		}
	}
	if !existDelLabel {
		fmt.Fprintln(l.Out, "    don't delete any labels")
	}
	fmt.Fprintln(l.Out, "")

	if existDelLabelWithIssue {
		fmt.Fprintln(l.Out, "  There is a label attached to issues in the delete labels.")
		fmt.Fprintln(l.Out, "  Please dettatch it from issues or write a label settings.")
		return nil
	}

	if !c.Bool("update") {
		return nil
	}

	fmt.Fprintln(l.Out, "  Update in progress...")
	for _, uOpe := range updOpes {
		// TODO github.Label doesn't have Description in v15.0.0, so don't set it...
		switch uOpe.Operation {
		case opeCrt:
			_, _, err = client.Issues.CreateLabel(context.Background(), *conf.User, *conf.Repo, &github.Label{Name: &uOpe.Name, Color: &uOpe.Color})
			if err != nil {
				fmt.Fprintf(l.Out, "    `%s` -> %s fail (err=\"%s\")\n", uOpe.Name, uOpe.Operation, err.Error())
			} else {
				fmt.Fprintf(l.Out, "    `%s` -> %s success\n", uOpe.Name, uOpe.Operation)
			}
		case opeUpd:
			_, _, err = client.Issues.EditLabel(context.Background(), *conf.User, *conf.Repo, uOpe.Name, &github.Label{Name: &uOpe.Name, Color: &uOpe.Color})
			if err != nil {
				fmt.Fprintf(l.Out, "    `%s` -> %s fail (err=\"%s\")\n", uOpe.Name, uOpe.Operation, err.Error())
			} else {
				fmt.Fprintf(l.Out, "    `%s` -> %s success\n", uOpe.Name, uOpe.Operation)
			}
		case opeDel:
			_, err = client.Issues.DeleteLabel(context.Background(), *conf.User, *conf.Repo, uOpe.Name)
			if err != nil {
				fmt.Fprintf(l.Out, "    `%s` -> %s fail (err=\"%s\")\n", uOpe.Name, uOpe.Operation, err.Error())
			} else {
				fmt.Fprintf(l.Out, "    `%s` -> %s success\n", uOpe.Name, uOpe.Operation)
			}
		case opeIss:
			for _, iNum := range uOpe.Issues {
				_, _, err = client.Issues.AddLabelsToIssue(context.Background(), *conf.User, *conf.Repo, iNum, []string{uOpe.Name})
				if err != nil {
					fmt.Fprintf(l.Out, "    `%s` -> %s fail (issun num = %d) (err=\"%s\")\n", uOpe.Name, uOpe.Operation, iNum, err.Error())
				} else {
					fmt.Fprintf(l.Out, "    `%s` -> %s success (issun num = %d)\n", uOpe.Name, uOpe.Operation, iNum)
				}
			}
		default:
			panic(fmt.Sprintf("undefine operation string \"%s\"", uOpe.Operation))
		}
	}

	return nil
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

func (l label) GetIssues(client *github.Client, user, repo, labelname string) ([]int, error) {

	opt := &github.IssueListByRepoOptions{
		State:     "open",
		Sort:      "created",
		Direction: "asc",
		Labels:    []string{labelname},
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	var issueNums []int
	for {
		issues, resp, err := client.Issues.ListByRepo(context.Background(), user, repo, opt)
		if err != nil {
			return nil, err
		}
		for _, is := range issues {
			issueNums = append(issueNums, *is.Number)
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return issueNums, nil
}

type updOpe struct {
	Name, Operation, Color, Desc string
	Issues                       []int
}

const (
	opeCrt = "create"
	opeUpd = "update"
	opeDel = "delete"
	opeIss = "add issues"
)

func (l label) CreateUpdOpe(name, operation, color, desc string, issues []int) updOpe {
	return updOpe{
		Name:      name,
		Operation: operation,
		Color:     color,
		Desc:      desc,
		Issues:    issues,
	}
}
