package main

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/urfave/cli"
)

var (
	issueCmd = cli.Command{
		Name:  "issue",
		Usage: "management related to issues or pull requests",
		Action: func(c *cli.Context) error {
			return action(c, issueMain)
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "except, e",
				Usage: "except issues attached \"pending\" labels",
			},
		},
	}
)

const (
	noAssigneesLabel = "(No Assignees)"
)

type issueInfo struct {
	TaskTable   taskAssignList
	Urgents     []int
	NoAssignees []int
	TaskCount   int
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

func (t taskAssignList) GetAssigneeList(userMap map[string]string) []string {
	l := []string{}
	for _, v := range t {
		l = append(l, getValueWithMap(v.Assignee, userMap))
	}
	return l
}

func issueMain(c *cli.Context, conf *config) error {

	cli := ghIssueClient{}
	cli.SetConf(conf)

	issues, err := getIssues(&cli)
	if err != nil {
		return err
	}

	exceptLabels := []string{}
	if c.Bool("except") {
		exceptLabels = conf.LabelConditions.Pendings
	}

	iInfo, err := createTaskTable(issues, conf.LabelConditions.Urgents, exceptLabels)
	if err != nil {
		return err
	}

	fmt.Print(getResultStr(iInfo, conf.Repo, conf.Message, exceptLabels, conf.UserMappings))

	return nil
}

func getIssues(cli ghIssueClientI) ([]ghIssue, error) {

	cli.AddValue("state", "open")
	cli.AddValue("per_page", "100")

	issues := []ghIssue{}
	for i := 1; i < 100; i++ {
		cli.AddValue("page", strconv.Itoa(i))
		tmpIssues, err := cli.GetIssues()
		if err != nil {
			return nil, err
		}
		if len(tmpIssues) <= 0 {
			break
		}
		issues = append(issues, tmpIssues...)
	}

	return issues, nil
}

func createTaskTable(issues []ghIssue, urgentLabels, exceptLabels []string) (issueInfo, error) {

	taskMap := make(map[string][]int)
	urgents := []int{}
	noAssignees := []int{}
	taskCount := 0

ISSUE_LOOP:
	for _, issue := range issues {
		if len(exceptLabels) > 0 {
			for _, label := range issue.Labels {
				if existStr(exceptLabels, label.Name) {
					continue ISSUE_LOOP
				}
			}
		}
		taskCount++
		if len(urgentLabels) > 0 {
			for _, label := range issue.Labels {
				if existStr(urgentLabels, label.Name) {
					urgents = append(urgents, issue.Number)
					break
				}
			}
		}
		if len(issue.Assignees) > 0 {
			for _, user := range issue.Assignees {
				if _, ok := taskMap[user.Name]; ok {
					taskMap[user.Name] = append(taskMap[user.Name], issue.Number)
				} else {
					taskMap[user.Name] = []int{issue.Number}
				}
			}
		} else {
			noAssignees = append(noAssignees, issue.Number)
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

func getResultStr(iInfo issueInfo, repo, message *string,
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
	rstStr += fmt.Sprintf("# Issue & PR List for `%s`\n", *repo)

	rstStr += fmt.Sprintf("\ttask count: %d\n", iInfo.TaskCount)
	rstStr += fmt.Sprintf("\turgent: %s\n", nvl(concatInt(iInfo.Urgents, ", ")))
	if len(exceptLabels) > 0 {
		rstStr += fmt.Sprintf("\texcepts labels: %s\n", nvl(concatStrWithBracket(exceptLabels, ", ", "`")))
	}
	rstStr += "\n```\n"
	for _, t := range iInfo.TaskTable {
		rstStr += createOneLine(getValueWithMap(t.Assignee, userMap), t.Tasks, iInfo, &maxLength)
	}

	if len(iInfo.NoAssignees) > 0 {
		rstStr += createOneLine(noAssigneesLabel, iInfo.NoAssignees, iInfo, &maxLength)
	}
	rstStr += "```\n"

	rstStr += fmt.Sprintf("\n%s\n", concatStrWith2Brackets(iInfo.TaskTable.GetAssigneeList(userMap), ", ", "@", ""))
	if message != nil {
		rstStr += fmt.Sprintln(*message)
	}

	return rstStr
}

func createOneLine(name string, tasks []int, iInfo issueInfo, maxLength *int) string {
	return fmt.Sprintf("- %s%s (%d): %s\n", name, space(*maxLength-len(name)), len(tasks), concatInt(tasks, ", "))
}
