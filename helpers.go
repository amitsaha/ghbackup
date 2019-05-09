package main

import (
	"context"
	"log"

	"github.com/google/go-github/v32/github"
	gitlab "github.com/xanzy/go-gitlab"
)

func getUsername(client interface{}, service string) string {

	if client == nil {
		log.Fatalf("Couldn't acquire a client to talk to %s", service)
	}

	if service == "github" {
		ctx := context.Background()
		user, _, err := client.(*github.Client).Users.Get(ctx, "")
		if err != nil {
			log.Fatal("Error retrieving username", err.Error())
		}
		return *user.Login
	}

	if service == "gitlab" {
		user, _, err := client.(*gitlab.Client).Users.CurrentUser()
		if err != nil {
			log.Fatal("Error retrieving username", err.Error())
		}
		return user.Username
	}

	return ""
}

func getGitHubOrgDetails(org string) *github.Organization {
	client := newClient("github", *gitHostURL)
	if client == nil {
		log.Fatalf("Couldn't acquire a client to talk to  gitlab")
	}
	ctx := context.Background()
	o, _, err := client.(*github.Client).Organizations.Get(ctx, org)
	if err != nil {
		log.Fatal("Error retrieving organization details", err.Error())
	}
	return o
}
