package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/urfave/cli"
)

const (
	noAssigneesLabel = "(No Assignees)"
	noPriorityLabel  = "(No Priority Labels)"
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
			cli.BoolFlag{
				Name:  "priority, p",
				Usage: "output priority list at the same time",
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
		exceptLabels = conf.getLabels("Low")
	}
	priorityLabels := []string{}
	if c.Bool("priority") {
		priorityLabels = conf.getPriorityLabels("")
	}

	iInfo := i.createIssueInfo(issues, conf.getLabels("High"), exceptLabels, priorityLabels)

	i.outputResult(iInfo, *conf.User, *conf.Repo, *conf.Message, exceptLabels, priorityLabels, conf.UserMappings)

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

func (i issue) createIssueInfo(baseIssues []*github.Issue, highLabels, exceptLabels, priorityLabels []string) issueInfo {

	issueAssignees := make(map[int][]string)
	assigneeIssues := make(map[string][]int)
	assignees := []string{}
	priorityIssues := make(map[string][]int)
	highIssues := []int{}
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

		// set issueAssignees, assigneeIssues and assignees
		issueAssignees[*issue.Number] = []string{}
		if len(issue.Assignees) > 0 {
			for _, user := range issue.Assignees {
				issueAssignees[*issue.Number] = append(issueAssignees[*issue.Number], *user.Login)
				if _, ok := assigneeIssues[*user.Login]; ok {
					assigneeIssues[*user.Login] = append(assigneeIssues[*user.Login], *issue.Number)
				} else {
					assigneeIssues[*user.Login] = []int{*issue.Number}
					assignees = append(assignees, *user.Login)
				}
			}
		} else {
			if _, ok := assigneeIssues[noAssigneesLabel]; ok {
				assigneeIssues[noAssigneesLabel] = append(assigneeIssues[noAssigneesLabel], *issue.Number)
			} else {
				assigneeIssues[noAssigneesLabel] = []int{*issue.Number}
			}
		}

		// set priorityIssues
		existPriority := false
		if len(priorityLabels) > 0 {
			for _, label := range issue.Labels {
				if existStr(priorityLabels, *label.Name) {
					if _, ok := priorityIssues[*label.Name]; ok {
						priorityIssues[*label.Name] = append(priorityIssues[*label.Name], *issue.Number)
					} else {
						priorityIssues[*label.Name] = []int{*issue.Number}
					}
					existPriority = true
				}
			}
			if !existPriority {
				if _, ok := priorityIssues[noPriorityLabel]; ok {
					priorityIssues[noPriorityLabel] = append(priorityIssues[noPriorityLabel], *issue.Number)
				} else {
					priorityIssues[noPriorityLabel] = []int{*issue.Number}
				}
			}
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
		return len(assigneeIssues[assignees[j]]) < len(assigneeIssues[assignees[i]])
	})
	sort.Ints(highIssues)

	return issueInfo{
		BaseIssues:      baseIssues,
		IssueAssignees:  issueAssignees,
		AssigneeIssues:  assigneeIssues,
		AssigneeRanking: assignees,
		PriorityIssues:  priorityIssues,
		HighIssues:      highIssues,
		ExceptIssueCnt:  exceptIssueCnt,
	}
}

func (i issue) outputResult(iInfo issueInfo, user, repo, message string,
	exceptLabels, priorityLabels []string, userMap userMappings) {

	if i.Out == nil {
		return
	}

	// prepare
	maxAssigneeLen := 0
	if _, ok := iInfo.AssigneeIssues[noAssigneesLabel]; ok {
		maxAssigneeLen = len(noAssigneesLabel)
	}
	for _, v := range iInfo.AssigneeRanking {
		if maxAssigneeLen < len(v) {
			maxAssigneeLen = len(v)
		}
	}

	maxPriorityLen := len(strconv.Itoa(*iInfo.BaseIssues[len(iInfo.BaseIssues)-1].Number))

	// output
	fmt.Fprintf(i.Out, "# Issue & PR List for `%s/%s`\n", user, repo)

	fmt.Fprintf(i.Out, "\ttask count: %d\n", len(iInfo.BaseIssues)-iInfo.ExceptIssueCnt)
	fmt.Fprintf(i.Out, "\turgent: %s\n", nvl(concatInt(iInfo.HighIssues, ", ")))
	if len(exceptLabels) > 0 {
		fmt.Fprintf(i.Out, "\texcepts labels: %s\n", nvl(concatStrWithBracket(exceptLabels, ", ", "`")))
	}
	// Assingee List
	if len(priorityLabels) > 0 {
		fmt.Fprintln(i.Out, "\n*Assingee List*\n```")
	}
	for _, v := range iInfo.AssigneeRanking {
		fmt.Fprint(i.Out, i.assigneeLine(userMap.getValue(v), iInfo.AssigneeIssues[v], maxAssigneeLen))
	}
	if _, ok := iInfo.AssigneeIssues[noAssigneesLabel]; ok {
		fmt.Fprint(i.Out, i.assigneeLine(noAssigneesLabel, iInfo.AssigneeIssues[noAssigneesLabel], maxAssigneeLen))
	}
	fmt.Fprintln(i.Out, "```")

	// Priority List
	if len(priorityLabels) > 0 {
		fmt.Fprintln(i.Out, "\n*Priority List*\n```")
		for _, v := range priorityLabels {
			if _, ok := iInfo.PriorityIssues[v]; !ok {
				continue
			}
			fmt.Fprintf(i.Out, "- %s\n", v)
			fmt.Fprint(i.Out, i.priorityLines(iInfo.PriorityIssues[v], maxPriorityLen, iInfo.IssueAssignees, userMap))
		}
		if _, ok := iInfo.PriorityIssues[noPriorityLabel]; ok {
			fmt.Fprintf(i.Out, "- %s\n", noPriorityLabel)
			fmt.Fprint(i.Out, i.priorityLines(iInfo.PriorityIssues[noPriorityLabel], maxPriorityLen, iInfo.IssueAssignees, userMap))
		}
		fmt.Fprintln(i.Out, "```")
	}

	fmt.Fprintf(i.Out, "\n%s\n", concatStrWith2Brackets(userMap.getValues(iInfo.AssigneeRanking), ", ", "@", ""))
	if message != "" {
		fmt.Fprintln(i.Out, message)
	}
}

func (i issue) assigneeLine(name string, issues []int, maxLen int) string {
	return fmt.Sprintf("- %s%s (%d): %s\n", name, space(maxLen-len(name)), len(issues), concatInt(issues, ", "))
}

func (i issue) priorityLines(issues []int, maxLen int, issueAssignees map[int][]string, userMap userMappings) string {
	str := ""
	for _, v := range issues {
		assignees := concatStr(userMap.getValues(issueAssignees[v]), ", ")
		if assignees == "" {
			assignees = noAssigneesLabel
		}
		str += fmt.Sprintf("  - %d%s: %s\n", v, space(maxLen-len(strconv.Itoa(v))), assignees)
	}

	return str
}

type issueInfo struct {
	BaseIssues      []*github.Issue
	IssueAssignees  map[int][]string
	AssigneeIssues  map[string][]int
	AssigneeRanking []string
	PriorityIssues  map[string][]int
	HighIssues      []int
	ExceptIssueCnt  int
}
