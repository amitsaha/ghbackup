package main

import (
	"context"
	"log"
	"net/url"
	"path"
	"strings"

	"github.com/google/go-github/v33/github"
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

func getDefaultGitbackupDir(service string) string {
	var gitbackupDir string

	homeDir, err := homedir.Dir()
	if err == nil {
		gitbackupDir = path.Join(homeDir, ".gitbackup")
	} else {
		log.Fatal("Could not determine home directory")
	}
	return gitbackupDir
}

func setupRepoDir(syncTarget string, workDir string, service string, githostURL string) string {
	var repoDir string

	if strings.HasPrefix(syncTarget, "gitlab:///") || strings.HasPrefix(syncTarget, "github:///") {
		if len(workDir) != 0 {
			repoDir = workDir
		} else {
			repoDir = getDefaultGitbackupDir(service)
		}
	} else if len(syncTarget) == 0 {
		repoDir = getDefaultGitbackupDir(service)
	} else {
		repoDir = syncTarget
	}

	if len(githostURL) == 0 {
		service = service + ".com"
		repoDir = path.Join(repoDir, service)
	} else {
		u, err := url.Parse(githostURL)
		if err != nil {
			panic(err)
		}
		repoDir = path.Join(repoDir, u.Host)
	}

	_, err := appFS.Stat(repoDir)
	if err != nil {
		log.Printf("%s doesn't exist, creating it\n", repoDir)
		err := appFS.MkdirAll(repoDir, 0771)
		if err != nil {
			log.Fatal(err)
		}
	}
	return repoDir
}
