package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/google/go-github/github"
	"github.com/urfave/cli"
)

const (
	noAssigneesLabel = "(No Assignees)"
)

func init() {
	cmdList = append(cmdList, cli.Command{
		Name:  "issue",
		Usage: "management related to issues or pull requests",
		Action: func(c *cli.Context) error {
			return action(c, &issue{Out: os.Stdout})
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "except, e",
				Usage: "except issues attached low level labels",
			},
		},
	})
}

type issue struct {
	Out io.Writer
}

func (i issue) Run(c *cli.Context, conf *config, client *github.Client) error {

	issues, err := i.getAllIssues(client, *conf.User, *conf.Repo)
	if err != nil {
		return err
	}

	exceptLabels := []string{}
	if c.Bool("except") {
		exceptLabels = conf.getLabel("Low")
	}

	iInfo, err := i.createIssueInfo(issues, conf.getLabel("High"), exceptLabels)
	if err != nil {
		return err
	}

	if i.Out != nil {
		fmt.Fprint(i.Out, i.getResultStr(iInfo, *conf.User, *conf.Repo, *conf.Message, exceptLabels, conf.UserMappings))
	}

	return nil
}

func (i issue) getAllIssues(client *github.Client, user, repo string) ([]*github.Issue, error) {

	opt := &github.IssueListByRepoOptions{
		State:     "open",
		Sort:      "created",
		Direction: "asc",
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	var allIssues []*github.Issue
	for {
		issues, resp, err := client.Issues.ListByRepo(context.Background(), user, repo, opt)
		if err != nil {
			return nil, err
		}
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allIssues, nil
}

func (i issue) createIssueInfo(baseIssues []*github.Issue, highLabels, exceptLabels []string) (issueInfo, error) {

	assigneeIssueMap := make(map[string][]int)
	assignees := []string{}
	highIssues := []int{}
	noAssigneeIssues := []int{}
	exceptIssueCnt := 0

ISSUE_LOOP:
	for _, issue := range baseIssues {

		// check except or not
		if len(exceptLabels) > 0 {
			for _, label := range issue.Labels {
				if existStr(exceptLabels, *label.Name) {
					exceptIssueCnt++
					continue ISSUE_LOOP
				}
			}
		}

		// set assigneeIssueMap and assignees
		if len(issue.Assignees) > 0 {
			for _, user := range issue.Assignees {
				if _, ok := assigneeIssueMap[*user.Login]; ok {
					assigneeIssueMap[*user.Login] = append(assigneeIssueMap[*user.Login], *issue.Number)
				} else {
					assigneeIssueMap[*user.Login] = []int{*issue.Number}
					assignees = append(assignees, *user.Login)
				}
			}
		} else {
			noAssigneeIssues = append(noAssigneeIssues, *issue.Number)
		}

		// set highIssues
		if len(highLabels) > 0 {
			for _, label := range issue.Labels {
				if existStr(highLabels, *label.Name) {
					highIssues = append(highIssues, *issue.Number)
					break
				}
			}
		}
	}

	// sort
	sort.Slice(assignees, func(i, j int) bool {
		return len(assigneeIssueMap[assignees[j]]) < len(assigneeIssueMap[assignees[i]])
	})
	sort.Ints(highIssues)
	sort.Ints(noAssigneeIssues)

	return issueInfo{
		BaseIssues:       baseIssues,
		AssigneeIssues:   assigneeIssueMap,
		AssigneeRanking:  assignees,
		HighIssues:       highIssues,
		NoAssigneeIssues: noAssigneeIssues,
		ExceptIssueCnt:   exceptIssueCnt,
	}, nil
}

func (i issue) getResultStr(iInfo issueInfo, user, repo, message string,
	exceptLabels []string, userMap userMappings) string {

	// prepare
	maxLength := 0
	if len(iInfo.NoAssigneeIssues) > 0 {
		maxLength = len(noAssigneesLabel)
	}
	for _, v := range iInfo.AssigneeRanking {
		if maxLength < len(v) {
			maxLength = len(v)
		}
	}

	// output
	var rstStr string
	rstStr += fmt.Sprintf("# Issue & PR List for `%s/%s`\n", user, repo)

	rstStr += fmt.Sprintf("\ttask count: %d\n", len(iInfo.BaseIssues)-iInfo.ExceptIssueCnt)
	rstStr += fmt.Sprintf("\turgent: %s\n", nvl(concatInt(iInfo.HighIssues, ", ")))
	if len(exceptLabels) > 0 {
		rstStr += fmt.Sprintf("\texcepts labels: %s\n", nvl(concatStrWithBracket(exceptLabels, ", ", "`")))
	}
	rstStr += "\n```\n"
	for _, v := range iInfo.AssigneeRanking {
		rstStr += i.createOneLine(userMap.getValue(v), iInfo.AssigneeIssues[v], &maxLength)
		// rstStr += i.createOneLine(v, iInfo.AssigneeIssues[v], &maxLength)
	}

	if len(iInfo.NoAssigneeIssues) > 0 {
		rstStr += i.createOneLine(noAssigneesLabel, iInfo.NoAssigneeIssues, &maxLength)
	}
	rstStr += "```\n"

	rstStr += fmt.Sprintf("\n%s\n", concatStrWith2Brackets(userMap.getValues(iInfo.AssigneeRanking), ", ", "@", ""))
	// rstStr += fmt.Sprintf("\n%s\n", concatStrWith2Brackets(iInfo.AssigneeRanking, ", ", "@", ""))
	if message != "" {
		rstStr += fmt.Sprintln(message)
	}

	return rstStr
}

func (i issue) createOneLine(name string, tasks []int, maxLength *int) string {
	return fmt.Sprintf("- %s%s (%d): %s\n", name, space(*maxLength-len(name)), len(tasks), concatInt(tasks, ", "))
}

type issueInfo struct {
	BaseIssues       []*github.Issue
	AssigneeIssues   map[string][]int
	AssigneeRanking  []string
	HighIssues       []int
	NoAssigneeIssues []int
	ExceptIssueCnt   int
}
