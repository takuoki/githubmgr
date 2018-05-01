package main

import (
	"fmt"
	"net/url"
)

const githubAPIDomain = "api.github.com"

type issue struct {
	Number    int     `json:"number"`
	Title     string  `json:"title"`
	State     string  `json:"state"`
	Labels    []label `json:"labels"`
	Assignees []user  `json:"assignees"`
}

type label struct {
	Name string `json:"name"`
}

type user struct {
	Name string `json:"login"`
}

func getIssueURL(conf *config, v url.Values) string {
	if conf.Token != nil {
		v.Add("access_token", *conf.Token)
	}
	return fmt.Sprintf("https://%s/repos/%s/%s/issues?%s", githubAPIDomain, *conf.User, *conf.Repo, v.Encode())
}
