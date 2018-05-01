package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/urfave/cli"
)

// Config is a configure for GitHub.
type config struct {
	User            *string         `json:"username"`
	Repo            *string         `json:"repository"`
	Token           *string         `json:"access_token"`
	Message         *string         `json:"message_to_assignee"`
	LabelConditions LabelConditions `json:"label_conditions"`
	UserMappingList []UserMapping   `json:"user_mappings"`
	UserMappings    map[string]string
}

// LabelConditions have some conditions related GitHub labels.
type LabelConditions struct {
	Urgents  []string `json:"urgents"`
	Pendings []string `json:"pendings"`
}

// UserMapping is a mapping username between GitHub and Slack.
type UserMapping struct {
	GitHubName string `json:"github_name"`
	SlackName  string `json:"slack_name"`
}

// Read is a function to get Config.
func readConfig(c *cli.Context) (*config, error) {

	conf := new(config)

	jsonStr, err := ioutil.ReadFile(c.GlobalString("config"))
	if err != nil {
		return conf, fmt.Errorf("not found config file (%s)", c.GlobalString("config"))
	}

	err = json.Unmarshal(jsonStr, conf)
	if err != nil {
		return conf, fmt.Errorf("something wrong in config file (%s)", c.GlobalString("config"))
	}

	// if a value is specified with a command line argument, use that value
	if str := c.GlobalString("user"); str != "" {
		conf.User = &str
	}
	if str := c.GlobalString("repo"); str != "" {
		conf.Repo = &str
	}
	if str := c.GlobalString("token"); str != "" {
		conf.Token = &str
	}

	// create UserMappings
	conf.UserMappings = make(map[string]string)
	for _, userMapping := range conf.UserMappingList {
		if _, exist := conf.UserMappings[userMapping.GitHubName]; exist {
			return nil, errors.New("duplicate github_name")
		}
		conf.UserMappings[userMapping.GitHubName] = userMapping.SlackName
	}

	// user and repo is mandatory
	if conf.User == nil {
		return conf, errors.New("username is mandatory")
	}
	if conf.Repo == nil {
		return conf, errors.New("repository name is mandatory")
	}

	return conf, nil
}
