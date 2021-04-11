package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/google/go-github/v34/github"
)

// MaxConcurrentClones is the upper limit of the maximum number of
// concurrent git clones
var MaxConcurrentClones = 20

var gitHostToken string
var useHTTPSClone *bool
var ignorePrivate *bool
var gitHostUsername string

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
	githostURL := flag.String("githost.url", "", "DNS of the custom Git host")
	backupDir := flag.String("backupdir", "", "Backup directory")
	ignorePrivate = flag.Bool("ignore-private", false, "Ignore private repositories/projects")
	ignoreFork := flag.Bool("ignore-fork", false, "Ignore repositories which are forks")
	useHTTPSClone = flag.Bool("use-https-clone", false, "Use HTTPS for cloning instead of SSH")

	// GitHub specific flags
	githubRepoType := flag.String("github.repoType", "all", "Repo types to backup (all, owner, public, private, member)")

	githubCreateUserMigration := flag.Bool("github.createUserMigration", false, "Download user data")

	githubListUserMigrations := flag.Bool("github.listUserMigrations", false, "List available user migrations")
	githubWaitForMigrationComplete := flag.Bool("github.waitForUserMigration", true, "Wait for migration to complete")

	// Gitlab specific flags
	gitlabRepoVisibility := flag.String("gitlab.projectVisibility", "all", "Visibility level of Projects to clone (internal, public, private, all)")
	gitlabProjectMembership := flag.String("gitlab.projectMembershipType", "all", "Project type to clone (all, owner, member)")

	flag.Parse()

	if len(*service) == 0 || !knownServices[*service] {
		log.Fatal("Please specify the git service type: github, gitlab")
	}

	if !validGitlabProjectMembership(*gitlabProjectMembership) {
		log.Fatal("Please specify a valid gitlab project membership - all/owner/member")
	}

	*backupDir = setupBackupDir(*backupDir, *service, *githostURL)

	client := newClient(*service, *githostURL)

	if *githubListUserMigrations {
		mList, err := getGithubUserMigrations(client)
		if err != nil {
			log.Fatal(err)
		}

		for _, m := range mList {
			mData, err := GetGithubUserMigration(client, m.ID)
			if err != nil {
				fmt.Printf("Error getting migration data: %v", *m.ID)
				// FIXME
				continue
			}

			var archiveURL string
			_, err = client.(*github.Client).Migrations.UserMigrationArchiveURL(context.Background(), *m.ID)
			if err != nil {
				archiveURL = "No Longer Available"
			} else {
				archiveURL = "Available for Download"
			}
			fmt.Printf("%v - %v - %v - %v\n", *mData.ID, *mData.CreatedAt, *mData.State, archiveURL)
		}

	} else if *githubCreateUserMigration {

		repos, err := getRepositories(client, *service, *githubRepoType, *gitlabRepoVisibility, *gitlabProjectMembership, *ignoreFork)
		if err != nil {
			log.Fatalf("Error getting list of repositories: %v", err)
		}

		log.Printf("Creating a user migration for %d repos", len(repos))
		m, err := createGithubUserMigration(context.Background(), client, repos)
		if err != nil {
			log.Fatalf("Error creating migration: %v", err)
		}
		if *githubWaitForMigrationComplete {
			downloadGithubUserMigrationData(client, *backupDir, m.ID)
		}

		orgs, err := getUserOwnedOrgs(client)
		if err != nil {
			log.Fatal("Error getting user organizations", err)
		}
		for _, o := range orgs {
			orgRepos, err := getGithubOrgRepositories(client, o)
			if err != nil {
				log.Fatal("Error getting org repos", err)
			}
			if len(orgRepos) == 0 {
				log.Printf("No repos found in %s", *o.Login)
				continue
			}
			log.Printf("Creating a org migration (%s) for %d repos", *o.Login, len(orgRepos))
			oMigration, err := createGithubOrgMigration(context.Background(), client, *o.Login, orgRepos)
			if err != nil {
				log.Fatalf("Error creating migration: %v", err)
			}
			if *githubWaitForMigrationComplete {
				downloadGithubOrgMigrationData(client, *o.Login, *backupDir, oMigration.ID)
			}
		}

	} else {
		tokens := make(chan bool, MaxConcurrentClones)
		gitHostUsername = getUsername(client, *service)

		if len(gitHostUsername) == 0 && !*ignorePrivate && *useHTTPSClone {
			log.Fatal("Your Git host's username is needed for backing up private repositories via HTTPS")
		}
		repos, err := getRepositories(client, *service, *githubRepoType, *gitlabRepoVisibility, *gitlabProjectMembership, *ignoreFork)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Printf("Backing up %v repositories now..\n", len(repos))
			for _, repo := range repos {
				tokens <- true
				wg.Add(1)
				go func(repo *Repository) {
					stdoutStderr, err := backUp(*backupDir, repo, &wg)
					if err != nil {
						log.Printf("Error backing up %s: %s\n", repo.Name, stdoutStderr)
					}
					<-tokens
				}(repo)
			}
		}
	}
}
