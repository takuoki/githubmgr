package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"

	"github.com/urfave/cli"
)

var (
	issueCmd = cli.Command{
		Name:  "issue",
		Usage: "management related to issues or pull requests",
		Action: func(c *cli.Context) error {
			return issueList(c)
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

func (t taskAssignList) GetAssigneeList() []string {
	l := []string{}
	for _, v := range t {
		l = append(l, v.Assignee)
	}
	return l
}

func issueList(c *cli.Context) error {

	conf, err := readConfig(c)
	if err != nil {
		return err
	}

	issues, err := getIssues(conf)
	if err != nil {
		return err
	}

	except := c.Bool("except")

	iInfo, err := createTaskTable(conf, issues, except)
	if err != nil {
		return err
	}

	return outputResult(conf, iInfo, except)
}

func getIssues(conf *config) ([]issue, error) {

	values := url.Values{}
	values.Add("state", "open")
	values.Add("per_page", "1000")

	rsp, err := http.Get(getIssueURL(conf, values))
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode != http.StatusOK {
		return nil, errors.New("cannot get issue list. please check URL information")
	}

	defer rsp.Body.Close()
	rspBody, _ := ioutil.ReadAll(rsp.Body)

	issues := []issue{}
	if err := json.Unmarshal(rspBody, &issues); err != nil {
		return nil, err
	}

	return issues, nil
}

func createTaskTable(conf *config, issues []issue, except bool) (issueInfo, error) {

	taskMap := make(map[string][]int)
	urgents := []int{}
	noAssignees := []int{}
	taskCount := 0

ISSUE_LOOP:
	for _, issue := range issues {
		if except {
			for _, label := range issue.Labels {
				if existStr(conf.LabelConditions.Pendings, label.Name) {
					continue ISSUE_LOOP
				}
			}
		}
		taskCount++
		for _, label := range issue.Labels {
			if existStr(conf.LabelConditions.Urgents, label.Name) {
				urgents = append(urgents, issue.Number)
			}
		}
		if len(issue.Assignees) > 0 {
			for _, user := range issue.Assignees {
				username, ok := conf.UserMappings[user.Name]
				if !ok {
					username = user.Name
				}
				if _, ok := taskMap[username]; ok {
					taskMap[username] = append(taskMap[username], issue.Number)
				} else {
					taskMap[username] = []int{issue.Number}
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
	assigneeList := []string{}
	for _, v := range taskTable {
		assigneeList = append(assigneeList, v.Assignee)
	}

	return issueInfo{
		TaskTable:   taskTable,
		Urgents:     urgents,
		NoAssignees: noAssignees,
		TaskCount:   taskCount,
	}, nil
}

func outputResult(conf *config, iInfo issueInfo, except bool) error {

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
	fmt.Printf("# Issue & PR List for `%s`\n", *conf.Repo)

	fmt.Printf("\ttask count: %d\n", iInfo.TaskCount)
	fmt.Printf("\turgent: %s\n", nvl(concatInt(iInfo.Urgents, ", ")))
	if except {
		fmt.Printf("\texcepts labels: %s\n", nvl(concatStrWithBracket(conf.LabelConditions.Pendings, ", ", "`")))
	}
	fmt.Println()

	fmt.Println("```")
	for _, t := range iInfo.TaskTable {
		fmt.Println(createOneLine(t.Assignee, t.Tasks, iInfo, &maxLength))
	}

	if len(iInfo.NoAssignees) > 0 {
		fmt.Println(createOneLine(noAssigneesLabel, iInfo.NoAssignees, iInfo, &maxLength))
	}
	fmt.Println("```")

	fmt.Printf("\n%s\n", concatStrWith2Brackets(iInfo.TaskTable.GetAssigneeList(), ", ", "@", ""))
	if conf.Message != nil {
		fmt.Println(*conf.Message)
	}

	return nil
}

func createOneLine(name string, tasks []int, iInfo issueInfo, maxLength *int) string {
	return fmt.Sprintf("- %s%s (%d): %s", name, space(*maxLength-len(name)), len(tasks), concatInt(tasks, ", "))
}
