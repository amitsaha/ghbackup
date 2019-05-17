package main

import (
	"flag"
	"log"
	"sync"
)

// MaxConcurrentClones is the upper limit of the maximum number of
// concurrent git clones
var MaxConcurrentClones = 20

var gitHostToken string
var gitlabToken string
var useHTTPSClone *bool
var ignorePrivate *bool
var skipForks *bool
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
	skipForks := flag.Bool("skip-forks", false, "Ignore repositories which are forks")
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

	workDir := setupWorkDir(*syncTarget, *service, *gitHostURL)

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
		// perform any filtering of repositories as desired
		repoFilter := RepositoryFilter{
			SkipForks: *skipForks,
		}
		repos = filterRepositories(repos, &repoFilter)
		log.Printf("Locally obtaining %v repositories now from %v.\n", len(repos), *service)
		for _, repo := range repos {
			tokens <- true
			wg.Add(1)
			go func(repo *Repository) {
				stdoutStderr, err := getRepo(workDir, repo, &wg)
				if err != nil {
					log.Printf("Error getting repo %s: %s\n", repo.Name, stdoutStderr)
				}
				<-tokens
			}(repo)
		}
		log.Printf("Syncing %v repositories now to %v.\n", len(repos), *syncTarget)
		for _, repo := range repos {
			tokens <- true
			wg.Add(1)
			go func(repo *Repository) {
				syncRepo(workDir, *syncTarget, repo, &wg)
				<-tokens
			}(repo)
		}
	}
}
