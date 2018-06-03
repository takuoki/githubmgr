package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/urfave/cli"
)

// Config is a configure for GitHub.
type config struct {
	User      *string `json:"username"`
	Repo      *string `json:"repository"`
	Token     *string `json:"access_token"`
	Message   *string `json:"message_to_assignee"`
	LabelRule struct {
		Priority []struct {
			LabelName *string `json:"label_name"`
			Level     *string `json:"level"`
		} `json:"priority"`
		Other []struct {
			LabelName *string `json:"label_name"`
			Level     *string `json:"level"`
		} `json:"other"`
	} `json:"label_rule"`
	UserMappingList []struct {
		GithubName *string `json:"github_name"`
		SlackName  *string `json:"slack_name"`
	} `json:"user_mappings"`
	UserMappings userMappings
}

func (c *config) getLabel(level string) []string {
	labels := []string{}
	for _, v := range c.LabelRule.Priority {
		if *v.Level == level {
			labels = append(labels, *v.LabelName)
		}
	}
	for _, v := range c.LabelRule.Other {
		if *v.Level == level {
			labels = append(labels, *v.LabelName)
		}
	}
	return labels
}

type userMappings map[string]string

func (u *userMappings) getValue(key string) string {
	if u == nil {
		return key
	}
	m := reflect.ValueOf(u).Elem()
	v := m.MapIndex(reflect.ValueOf(key))
	if v == (reflect.Value{}) {
		return key
	}
	return v.String()
}

func (u *userMappings) getValues(keys []string) []string {
	values := []string{}
	for _, k := range keys {
		values = append(values, u.getValue(k))
	}
	return values
}

func readConfig(c *cli.Context) (*config, error) {

	conf := new(config)

	jsonStr, err := ioutil.ReadFile(c.GlobalString("config"))
	if err != nil {
		return nil, fmt.Errorf("not found config file (%s)", c.GlobalString("config"))
	}

	err = json.Unmarshal(jsonStr, conf)
	if err != nil {
		return nil, fmt.Errorf("something wrong in config file (%s)", c.GlobalString("config"))
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
		if _, exist := conf.UserMappings[*userMapping.GithubName]; exist {
			return nil, errors.New("duplicate github_name")
		}
		conf.UserMappings[*userMapping.GithubName] = *userMapping.SlackName
	}

	// user and repo is mandatory
	if conf.User == nil {
		return nil, errors.New("username is mandatory")
	}
	if conf.Repo == nil {
		return nil, errors.New("repository name is mandatory")
	}

	return conf, nil
}
