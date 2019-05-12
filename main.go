package main

import (
	"flag"
	"log"
	"strings"
	"sync"
)

// MaxConcurrentClones is the upper limit of the maximum number of
// concurrent git clones
var MaxConcurrentClones = 20

var gitHostToken string
var gitlabToken string
var useHTTPSClone *bool
var ignorePrivate *bool
var ignoreForks *bool
var gitHostUsername string
var gitHostURL *string
var cleanSync *bool

func main() {

	// Used for waiting for all the goroutines to finish before exiting
	var wg sync.WaitGroup
	defer wg.Wait()

	// The services we know of
	knownServices := map[string]bool{
		"github": true,
		"gitlab": true,
	}

	// Generic flags
	service := flag.String("service", "", "Git Hosted Service Name (github/gitlab)")
	gitHostURL = flag.String("githost.url", "", "DNS of the custom Git host")
	syncTarget := flag.String("target", "", "Sync target")
	ignorePrivate = flag.Bool("ignore-private", false, "Ignore private repositories/projects")
	useHTTPSClone = flag.Bool("use-https-clone", false, "Use HTTPS for cloning instead of SSH")
	ignoreForks = flag.Bool("ignore-forks", false, "Ignore repositories which are forks")
	cleanSync = flag.Bool("clean-sync", false, "Recreate repositories on sync")

	// GitHub specific flags
	githubRepoType := flag.String("github.repoType", "all", "Repo types to backup (all, owner, member)")

	// Gitlab specific flags
	gitlabRepoVisibility := flag.String("gitlab.projectVisibility", "internal", "Visibility level of Projects to clone (internal, public, private)")
	gitlabProjectMembership := flag.String("gitlab.projectMembershipType", "all", "Project type to clone (all, owner, member)")

	flag.Parse()

	if len(*service) == 0 || !knownServices[*service] {
		log.Fatal("Please specify the git service type: github, gitlab")
	}

	var backupDir string
	if len(*syncTarget) == 0 || strings.HasPrefix(*syncTarget, "file:///") {
		backupDir = setupBackupDir(*syncTarget, *service, *gitHostURL)
	} else {
		backupDir = *syncTarget
	}

	tokens := make(chan bool, MaxConcurrentClones)
	client := newClient(*service, *gitHostURL)

	gitHostUsername = getUsername(client, *service)

	if len(gitHostUsername) == 0 && !*ignorePrivate && *useHTTPSClone {
		log.Fatal("Your Git host's username is needed for backing up private repositories via HTTPS")
	}
	repos, err := getRepositories(client, *service, *githubRepoType, *gitlabRepoVisibility, *gitlabProjectMembership)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Backing up %v repositories now..\n", len(repos))
		for _, repo := range repos {
			if repo.Fork && *ignoreForks {
				continue
			}
			tokens <- true
			wg.Add(1)
			go func(repo *Repository) {
				stdoutStderr, err := backUp(backupDir, repo, &wg)
				if err != nil {
					log.Printf("Error backing up %s: %s\n", repo.Name, stdoutStderr)
				}
				<-tokens
			}(repo)
		}
	}
}
