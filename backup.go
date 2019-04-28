package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"path"
	"strings"
	"sync"

	gitlab "github.com/xanzy/go-gitlab"

	"github.com/google/go-github/github"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
)

// We have them here so that we can override these in the tests
var execCommand = exec.Command
var appFS = afero.NewOsFs()
var gitCommand = "git"

// Check if we have a copy of the repo already, if
// we do, we update the repo, else we do a fresh clone
func backUp(client interface{}, backupDir string, repo *Repository, wg *sync.WaitGroup) ([]byte, error) {
	defer wg.Done()

	if strings.HasPrefix(backupDir, "gitlab:///") || strings.HasPrefix(backupDir, "github:///") {
		backupDir = "/tmp/gitbackupworkspace"
	}

	repoDir := path.Join(backupDir, repo.Namespace, repo.Name)
	_, err := appFS.Stat(repoDir)

	var stdoutStderr []byte
	if err == nil {
		log.Printf("%s exists, updating. \n", repo.Name)
		cmd := execCommand(gitCommand, "-C", repoDir, "pull")
		stdoutStderr, err = cmd.CombinedOutput()
	} else {
		log.Printf("Cloning %s\n", repo.Name)
		log.Printf("%#v\n", repo)

		if repo.Private && useHTTPSClone != nil && *useHTTPSClone && ignorePrivate != nil && !*ignorePrivate {
			// Add username and token to the clone URL
			// https://gitlab.com/amitsaha/testproject1 => https://amitsaha:token@gitlab.com/amitsaha/testproject1
			u, err := url.Parse(repo.CloneURL)
			if err != nil {
				log.Fatalf("Invalid clone URL: %v\n", err)
			}
			repo.CloneURL = u.Scheme + "://" + gitHostUsername + ":" + gitHostToken + "@" + u.Host + u.Path
		}

		cmd := execCommand(gitCommand, "clone", repo.CloneURL, repoDir)
		stdoutStderr, err = cmd.CombinedOutput()
	}

	if strings.HasPrefix(backupDir, "gitlab:///") {
		handleSyncGitlab(client, repo, backupDir)
	}
	if strings.HasPrefix(backupDir, "github:///") {
		handleSyncGithub(client, repo, backupDir)
	}

	return stdoutStderr, err
}

func setupBackupDir(backupDir string, service string, githostURL string) string {
	if len(backupDir) == 0 {
		homeDir, err := homedir.Dir()
		if err == nil {
			service = service + ".com"
			backupDir = path.Join(homeDir, ".gitbackup", service)
		} else {
			log.Fatal("Could not determine home directory and backup directory not specified")
		}
	} else {
		if len(githostURL) == 0 {
			service = service + ".com"
			backupDir = path.Join(backupDir, service)
		} else {
			u, err := url.Parse(githostURL)
			if err != nil {
				panic(err)
			}
			backupDir = path.Join(backupDir, u.Host)
		}
	}
	_, err := appFS.Stat(backupDir)
	if err != nil {
		log.Printf("%s doesn't exist, creating it\n", backupDir)
		err := appFS.MkdirAll(backupDir, 0771)
		if err != nil {
			log.Fatal(err)
		}
	}
	return backupDir
}

func handleSyncGitlab(client interface{}, repo *Repository, target string) {

	if client == nil {
		log.Fatalf("Couldn't acquire a client to talk to  gitlab")
	}

	projectName := fmt.Sprintf("%s/%s", repo.Namespace, repo.Name)
	project, _, err := client.(*gitlab.Client).Projects.GetProject(projectName, nil)
	if err != nil {
		log.Fatal("Error retrieving username", err.Error())
	}
	fmt.Printf("%v\n", project)
}

func handleSyncGithub(client interface{}, repo *Repository, target string) {
	if client == nil {
		log.Fatalf("Couldn't acquire a client to talk to github")
	}

	ctx := context.Background()
	user, _, err := client.(*github.Client).Users.Get(ctx, "")
	if err != nil {
		log.Fatal("Error retrieving username", err.Error())
	}

}
