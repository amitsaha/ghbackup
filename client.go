package main

import (
	"log"
	"net/url"
	"os"

	"github.com/google/go-github/v32/github"
	gitlab "github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"
)

func newClient(service string, gitHostURL string) interface{} {
	var gitHostURLParsed *url.URL
	var err error

	// If a git host URL has been passed in, we assume it's
	// a gitlab installation
	if len(gitHostURL) != 0 {
		gitHostURLParsed, err = url.Parse(gitHostURL)
		if err != nil {
			log.Fatalf("Invalid gitlab URL: %s", gitHostURL)
		}
		api, _ := url.Parse("api/v4/")
		gitHostURLParsed = gitHostURLParsed.ResolveReference(api)
	}

	if service == "github" {
		githubToken := os.Getenv("GITHUB_TOKEN")
		if githubToken == "" {
			log.Fatal("GITHUB_TOKEN environment variable not set")
		}
		gitHostToken = githubToken
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		tc := oauth2.NewClient(oauth2.NoContext, ts)
		client := github.NewClient(tc)
		if gitHostURLParsed != nil {
			client.BaseURL = gitHostURLParsed
		}
		return client
	}

	if service == "gitlab" {
		gitlabToken = os.Getenv("GITLAB_TOKEN")
		if gitlabToken == "" {
			log.Fatal("GITLAB_TOKEN environment variable not set")
		}
		gitHostToken = gitlabToken

		var client *gitlab.Client
		var err error

		if gitHostURLParsed != nil {
			client, err = gitlab.NewClient(gitlabToken, gitlab.WithBaseURL(gitHostURL))
		} else {
			client, err = gitlab.NewClient(gitlabToken)
		}
		if err != nil {
			log.Fatal("Failed to create gitlab client")
		}

		return client
	}
	return nil
}
