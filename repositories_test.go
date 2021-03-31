package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-github/v32/github"
	gitlab "github.com/xanzy/go-gitlab"
)

var (
	GitHubClient *github.Client
	GitLabClient *gitlab.Client
	mux          *http.ServeMux
	server       *httptest.Server
)

func setup() {
	os.Setenv("GITHUB_TOKEN", "$$$randome")
	os.Setenv("GITLAB_TOKEN", "$$$randome")

	// test server
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
	base, _ := url.Parse(server.URL)

	// Add a trailing slash because GitHub SDK expects it
	u, err := url.Parse("/")
	if err != nil {
		log.Fatal(err)
	}
	url := base.ResolveReference(u)

	// github client configured to use test server
	GitHubClient = github.NewClient(nil)
	GitHubClient.BaseURL = url

	// github client configured to use test server
	GitLabClient = gitlab.NewClient(nil, "")
	GitLabClient.SetBaseURL(url.String())
}

func teardown() {
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("GITLAB_TOKEN")
	server.Close()
}

func TestGetPublicGitHubRepositories(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/user/repos", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"full_name": "test/r1", "id":1, "ssh_url": "https://github.com/u/r1", "name": "r1", "private": false}]`)
	})

	repos, err := getRepositories(GitHubClient, "github", "all", "", "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	var expected []*Repository
	expected = append(expected, &Repository{Namespace: "test", CloneURL: "https://github.com/u/r1", Name: "r1", Private: false})
	if !reflect.DeepEqual(repos, expected) {
		t.Errorf("Expected %+v, Got %+v", expected, repos)
	}
}

func TestGetPrivateGitHubRepositories(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/user/repos", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"full_name": "test/r1", "id":1, "ssh_url": "https://github.com/u/r1", "name": "r1", "private": true}]`)
	})

	repos, err := getRepositories(GitHubClient, "github", "all", "", "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	var expected []*Repository
	expected = append(expected, &Repository{Namespace: "test", CloneURL: "https://github.com/u/r1", Name: "r1", Private: true})
	if !reflect.DeepEqual(repos, expected) {
		t.Errorf("Expected %+v, Got %+v", expected, repos)
	}
}

func TestGetGitLabRepositories(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/api/v4/projects", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"path_with_namespace": "test/r1", "id":1, "ssh_url_to_repo": "https://gitlab.com/u/r1", "name": "r1"}]`)
	})

	repos, err := getRepositories(GitLabClient, "gitlab", "internal", "", "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	var expected []*Repository
	expected = append(expected, &Repository{Namespace: "test", CloneURL: "https://gitlab.com/u/r1", Name: "r1"})
	if !reflect.DeepEqual(repos, expected) {
		for i := 0; i < len(repos); i++ {
			t.Errorf("Expected %+v, Got %+v", expected[i], repos[i])
		}
	}
}
