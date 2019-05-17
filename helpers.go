package main

import (
	"context"
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/google/go-github/v32/github"
	gitlab "github.com/xanzy/go-gitlab"

	homedir "github.com/mitchellh/go-homedir"
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

func setupWorkDir(syncTarget string, service string, githostURL string) string {
	var workDir string

	if len(syncTarget) == 0 || strings.HasPrefix(syncTarget, "gitlab:///") || strings.HasPrefix(syncTarget, "github:///") {
		homeDir, err := homedir.Dir()
		if err == nil {
			service = service + ".com"
			workDir = path.Join(homeDir, ".gitbackup", service)
		} else {
			log.Fatal("Could not determine home directory and target directory not specified")
		}
	} else {
		if len(githostURL) == 0 {
			service = service + ".com"
			workDir = path.Join(syncTarget, service)
		} else {
			u, err := url.Parse(githostURL)
			if err != nil {
				panic(err)
			}
			workDir = path.Join(syncTarget, u.Host)
		}
	}
	_, err := appFS.Stat(workDir)
	if err != nil {
		log.Printf("%s doesn't exist, creating it\n", workDir)
		err := appFS.MkdirAll(workDir, 0771)
		if err != nil {
			log.Fatal(err)
		}
	}
	return workDir
}
