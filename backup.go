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
func backUp(backupDir string, repo *Repository, wg *sync.WaitGroup) ([]byte, error) {
	defer wg.Done()

	var syncLocation string

	if strings.HasPrefix(backupDir, "gitlab:///") || strings.HasPrefix(backupDir, "github:///") {
		syncLocation = backupDir
		backupDir = "/tmp/gitbackupworkspace"
	}
	log.Printf("backupdir: %v\n", backupDir)

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

	if strings.HasPrefix(syncLocation, "gitlab:///") {
		handleSyncGitlab(repo, backupDir, syncLocation)
	}
	if strings.HasPrefix(syncLocation, "github:///") {
		handleSyncGithub(repo, backupDir, syncLocation)
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

func handleSyncGitlab(repo *Repository, workspace string, target string) {

	//FIXME: gitlab username != github username

	client := newClient("gitlab", *gitHostURL)
	if client == nil {
		log.Fatalf("Couldn't acquire a client to talk to  gitlab")
	}

	projectName := fmt.Sprintf("%s/%s", repo.Namespace, repo.Name)
	project, resp, err := client.(*gitlab.Client).Projects.GetProject(projectName, nil)
	if err != nil && resp.StatusCode != 404 {
		log.Fatal("Error checking if project exists", err.Error())
	}

	// FIXME: move this to somewhere else so that we don't do it for every backup
	// operation
	gitlabUsername := getUsername(client, "gitlab")

	if resp.StatusCode == 404 {

		log.Printf("Creating project in gitlab: %s\n", projectName)
		log.Printf(repo.Namespace)
		log.Printf(gitHostUsername)

		// check if namespace is not a username and if it doesn't exist
		// create it
		if repo.Namespace != gitlabUsername {
			_, resp, err := client.(*gitlab.Client).Groups.GetGroup(repo.Namespace)
			if err != nil && resp.StatusCode != 404 {
				log.Fatal("Error checking if group exists", err.Error())
			}
			if resp.StatusCode == 404 {
				log.Printf("Creating group in gitlab: %s\n", repo.Namespace)
				// if the org has any private repos, default to a private group

				// FIXME: release notes, perhaps a paramater?
				githubOrgDetails := getGitHubOrgDetails(repo.Namespace)
				var visibility gitlab.VisibilityValue
				if *githubOrgDetails.TotalPrivateRepos > 0 {
					visibility = gitlab.PrivateVisibility
				} else {
					visibility = gitlab.PublicVisibility
				}

				groupDesc := fmt.Sprintf("Imported from github %s", repo.Namespace)

				// future work
				lfsEnabled := false

				group := gitlab.CreateGroupOptions{
					Name:        &repo.Namespace,
					Path:        &repo.Namespace,
					Visibility:  &visibility,
					Description: &groupDesc,
					LFSEnabled:  &lfsEnabled,
				}
				g, _, err := client.(*gitlab.Client).Groups.CreateGroup(&group)
				if err != nil {
					log.Fatal("Error creating group", err.Error())
				}
				log.Printf("GitLab group created: %v\n", g)

			}
		}
		var namespace *gitlab.Namespace
		namespace, _, err := client.(*gitlab.Client).Namespaces.GetNamespace(repo.Namespace)
		if err != nil {
			log.Fatal("Error querying namespace", err.Error())
		}
		var repoVisibility gitlab.VisibilityValue
		if repo.Private {
			repoVisibility = gitlab.PrivateVisibility
		} else {
			repoVisibility = gitlab.PublicVisibility
		}
		// create project
		pCreateOptions := gitlab.CreateProjectOptions{
			Name:        &repo.Name,
			Path:        &repo.Name,
			NamespaceID: &namespace.ID,
			Visibility:  &repoVisibility,
		}
		p, _, err := client.(*gitlab.Client).Projects.CreateProject(&pCreateOptions)
		if err != nil {
			log.Fatal("Error creating project in GitLab", err.Error())
		}
		project = p
	}
	// add remote
	repoDir := path.Join(workspace, repo.Namespace, repo.Name)
	var stdoutStderr []byte

	// Add username and token to the clone URL
	// https://gitlab.com/amitsaha/testproject1 => https://amitsaha:token@gitlab.com/amitsaha/testproject1
	u, err := url.Parse(project.HTTPURLToRepo)
	if err != nil {
		log.Fatalf("Invalid clone URL: %v\n", err)
	}
	remoteURL := u.Scheme + "://" + gitlabUsername + ":" + gitlabToken + "@" + u.Host + u.Path

	// check if remote exists, if yes, use git remote set-url
	// else add remote
	cmd := execCommand(gitCommand, "-C", repoDir, "remote")
	stdoutStderr, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatal("Error listing remotes", string(stdoutStderr))
	}
	// FIXME: Windows suport
	var gitlabRemoteExists bool
	for _, remote := range strings.Split(string(stdoutStderr), "\n") {
		if remote == "gitlab" {
			gitlabRemoteExists = true
		}
	}
	if gitlabRemoteExists {
		cmd := execCommand(gitCommand, "-C", repoDir, "remote", "set-url", "gitlab", remoteURL)
		stdoutStderr, err = cmd.CombinedOutput()
		if err != nil {
			log.Fatal("Error setting GitLab remote url", string(stdoutStderr))
		}
	} else {
		cmd := execCommand(gitCommand, "-C", repoDir, "remote", "add", "gitlab", remoteURL)
		stdoutStderr, err = cmd.CombinedOutput()
		if err != nil {
			log.Fatal("Error adding GitLab remote", string(stdoutStderr))
		}
	}

	cmd = execCommand(gitCommand, "-C", repoDir, "push", "gitlab")
	stdoutStderr, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatal("Error pushing to gitlab", string(stdoutStderr))
	}

}

func handleSyncGithub(repo *Repository, workspace string, target string) {
	client := newClient("github", *gitHostURL)
	if client == nil {
		log.Fatalf("Couldn't acquire a client to talk to github")
	}

	owner := repo.Namespace
	repoName := repo.Name
	ctx := context.Background()
	r, resp, err := client.(*github.Client).Repositories.Get(ctx, owner, repoName)
	if err != nil && resp.StatusCode != 404 {
		log.Fatal("Error checking if project exists", err.Error())
	}

	if resp.StatusCode == 404 {
		log.Printf("Creating repo in github: %s/%s\n", owner, repoName)
	}
	fmt.Printf("repo exists: %v\n", r)
}
