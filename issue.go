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

	iInfo, err := i.createTaskTable(issues, conf.getLabel("High"), exceptLabels)
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

func (i issue) createTaskTable(issues []*github.Issue, urgentLabels, exceptLabels []string) (issueInfo, error) {

	taskMap := make(map[string][]int)
	urgents := []int{}
	noAssignees := []int{}
	taskCount := 0

ISSUE_LOOP:
	for _, issue := range issues {
		if len(exceptLabels) > 0 {
			for _, label := range issue.Labels {
				if existStr(exceptLabels, *label.Name) {
					continue ISSUE_LOOP
				}
			}
		}
		taskCount++
		if len(urgentLabels) > 0 {
			for _, label := range issue.Labels {
				if existStr(urgentLabels, *label.Name) {
					urgents = append(urgents, *issue.Number)
					break
				}
			}
		}
		if len(issue.Assignees) > 0 {
			for _, user := range issue.Assignees {
				if _, ok := taskMap[*user.Login]; ok {
					taskMap[*user.Login] = append(taskMap[*user.Login], *issue.Number)
				} else {
					taskMap[*user.Login] = []int{*issue.Number}
				}
			}
		} else {
			noAssignees = append(noAssignees, *issue.Number)
		}
	}

	// sort
	sort.Ints(urgents)
	sort.Ints(noAssignees)
	var taskTable taskAssignList = []taskAssign{}
	for k, v := range taskMap {
		sort.Ints(v)
		taskTable = append(taskTable, taskAssign{Assignee: k, Tasks: v})
	}
	sort.Sort(taskTable)

	return issueInfo{
		TaskTable:   taskTable,
		Urgents:     urgents,
		NoAssignees: noAssignees,
		TaskCount:   taskCount,
	}, nil
}

func (i issue) getResultStr(iInfo issueInfo, user, repo, message string,
	exceptLabels []string, userMap map[string]string) string {

	// prepare
	maxLength := 0
	if len(iInfo.NoAssignees) > 0 {
		maxLength = len(noAssigneesLabel)
	}
	for _, t := range iInfo.TaskTable {
		if maxLength < len(t.Assignee) {
			maxLength = len(t.Assignee)
		}
	}

	// output
	var rstStr string
	rstStr += fmt.Sprintf("# Issue & PR List for `%s/%s`\n", user, repo)

	rstStr += fmt.Sprintf("\ttask count: %d\n", iInfo.TaskCount)
	rstStr += fmt.Sprintf("\turgent: %s\n", nvl(concatInt(iInfo.Urgents, ", ")))
	if len(exceptLabels) > 0 {
		rstStr += fmt.Sprintf("\texcepts labels: %s\n", nvl(concatStrWithBracket(exceptLabels, ", ", "`")))
	}
	rstStr += "\n```\n"
	for _, t := range iInfo.TaskTable {
		rstStr += i.createOneLine(getValueWithMap(t.Assignee, userMap), t.Tasks, iInfo, &maxLength)
	}

	if len(iInfo.NoAssignees) > 0 {
		rstStr += i.createOneLine(noAssigneesLabel, iInfo.NoAssignees, iInfo, &maxLength)
	}
	rstStr += "```\n"

	rstStr += fmt.Sprintf("\n%s\n", concatStrWith2Brackets(iInfo.TaskTable.GetAssigneeList(userMap), ", ", "@", ""))
	if message != "" {
		rstStr += fmt.Sprintln(message)
	}

	return rstStr
}

func (i issue) createOneLine(name string, tasks []int, iInfo issueInfo, maxLength *int) string {
	return fmt.Sprintf("- %s%s (%d): %s\n", name, space(*maxLength-len(name)), len(tasks), concatInt(tasks, ", "))
}

type taskAssign struct {
	Assignee string
	Tasks    []int
}

type taskAssignList []taskAssign

func (t taskAssignList) Len() int {
	return len(t)
}

func (t taskAssignList) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t taskAssignList) Less(i, j int) bool {
	return len(t[i].Tasks) > len(t[j].Tasks)
}

type issueInfo struct {
	TaskTable   taskAssignList
	Urgents     []int
	NoAssignees []int
	TaskCount   int
}

func (t taskAssignList) GetAssigneeList(userMap map[string]string) []string {
	l := []string{}
	for _, v := range t {
		l = append(l, getValueWithMap(v.Assignee, userMap))
	}
	return l
}
