package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const githubAPIDomain = "api.github.com"

// json format
type ghIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Labels    []ghLabel `json:"labels"`
	Assignees []ghUser  `json:"assignees"`
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghUser struct {
	Name string `json:"login"`
}

// client
type ghClientI interface {
	AddValue(string, string)
}

type ghClient struct {
	Conf   *config
	Values url.Values
}

func (c *ghClient) SetConf(conf *config) {
	c.Conf = conf
}

func (c *ghClient) AddValue(key, value string) {
	if c.Values == nil {
		c.Values = url.Values{}
	}
	c.Values.Add(key, value)
}

func (c *ghClient) GetValues() url.Values {
	return c.Values
}

func (c *ghClient) Get(url string) ([]byte, error) {
	rsp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if rsp.StatusCode != http.StatusOK {
		return nil, errors.New("cannot get issue list. please check URL information")
	}

	defer rsp.Body.Close()
	return ioutil.ReadAll(rsp.Body)
}

type ghIssueClientI interface {
	ghClientI
	GetIssues() ([]ghIssue, error)
}

type ghIssueClient struct {
	ghClient
}

func (c *ghIssueClient) GetURL() string {
	if c.Conf == nil {
		panic("configuration of github client is nil")
	}
	if c.Conf.Token != nil {
		c.AddValue("access_token", *c.Conf.Token)
	}
	return fmt.Sprintf("https://%s/repos/%s/%s/issues?%s", githubAPIDomain,
		*c.Conf.User, *c.Conf.Repo, c.GetValues().Encode())
}

func (c *ghIssueClient) GetIssues() ([]ghIssue, error) {

	rsp, err := c.Get(c.GetURL())
	if err != nil {
		return nil, err
	}

	issues := []ghIssue{}
	if err := json.Unmarshal(rsp, &issues); err != nil {
		return nil, err
	}

	return issues, nil
}
