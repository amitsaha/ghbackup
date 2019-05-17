package main

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/v32/github"
	gitlab "github.com/xanzy/go-gitlab"
)

// Response is derived from the following sources:
// https://github.com/google/go-github/blob/27c7c32b6d369610435bd2ad7b4d8554f235eb01/github/github.go#L301
// https://github.com/xanzy/go-gitlab/blob/3acf8d75e9de17ad4b41839a7cabbf2537760ab4/gitlab.go#L286
type Response struct {
	*http.Response

	// These fields provide the page values for paginating through a set of
	// results.  Any or all of these may be set to the zero value for
	// responses that are not part of a paginated set, or for which there
	// are no additional pages.

	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int
}

// Repository is a container for the details for a repository
// we will backup
type Repository struct {
	CloneURL  string
	Name      string
	Namespace string
	Fork      bool
	Private   bool
}

// RepositoryFilter is a container for any filters to apply
// on the obtained repository list
type RepositoryFilter struct {
	SkipForks bool
}

func getRepositories(client interface{}, service string, githubRepoType string, gitlabRepoVisibility string, gitlabProjectType string) ([]*Repository, error) {

	if client == nil {
		log.Fatalf("Couldn't acquire a client to talk to %s", service)
	}

	var repositories []*Repository
	var cloneURL string

	if service == "github" {
		ctx := context.Background()
		options := github.RepositoryListOptions{Type: githubRepoType}
		for {
			repos, resp, err := client.(*github.Client).Repositories.List(ctx, "", &options)
			if err == nil {
				for _, repo := range repos {
					namespace := strings.Split(*repo.FullName, "/")[0]
					if useHTTPSClone != nil && *useHTTPSClone {
						cloneURL = *repo.CloneURL
					} else {
						cloneURL = *repo.SSHURL
					}
					repositories = append(repositories, &Repository{CloneURL: cloneURL, Name: *repo.Name, Namespace: namespace, Fork: *repo.Fork, Private: *repo.Private})
				}
			} else {
				return nil, err
			}
			if resp.NextPage == 0 {
				break
			}
			options.ListOptions.Page = resp.NextPage
		}
	}

	if service == "gitlab" {
		var visibility gitlab.VisibilityValue
		var options gitlab.ListProjectsOptions

		var owned bool
		var memberOf bool

		if gitlabProjectType == "owner" {
			owned = true
		}
		if gitlabProjectType == "member" {
			memberOf = true
		}

		if gitlabProjectType == "all" {
			owned = true
			memberOf = true
		}

		if gitlabRepoVisibility != "all" {
			switch gitlabRepoVisibility {
			case "public":
				visibility = gitlab.PublicVisibility
			case "private":
				visibility = gitlab.PrivateVisibility
			case "internal":
				fallthrough
			case "default":
				visibility = gitlab.InternalVisibility
			}
			options = gitlab.ListProjectsOptions{Visibility: &visibility, Membership: &memberOf, Owned: &owned}
		} else {
			options = gitlab.ListProjectsOptions{Membership: &memberOf, Owned: &owned}
		}

		for {
			repos, resp, err := client.(*gitlab.Client).Projects.ListProjects(&options)
			if err == nil {
				for _, repo := range repos {
					namespace := strings.Split(repo.PathWithNamespace, "/")[0]
					if useHTTPSClone != nil && *useHTTPSClone {
						cloneURL = repo.WebURL
					} else {
						cloneURL = repo.SSHURLToRepo
					}
					repositories = append(repositories, &Repository{CloneURL: cloneURL, Name: repo.Name, Namespace: namespace, Private: repo.Public, Fork: repo.ForkedFromProject != nil})
				}
			} else {
				return nil, err
			}
			if resp.NextPage == 0 {
				break
			}
			options.ListOptions.Page = resp.NextPage
		}
	}

	return repositories, nil
}

func filterRepositories(repos []*Repository, repoFilter *RepositoryFilter) []*Repository {
	var fRepos []*Repository

	// skip forks if specified
	for _, repo := range repos {
		if repo.Fork && repoFilter.SkipForks {
			continue
		}
		fRepos = append(fRepos, repo)
	}
	return fRepos
}
