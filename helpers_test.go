package main

import (
	"log"
	"path"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
)

func TestSetupWorkDi(t *testing.T) {
	appFS = afero.NewMemMapFs()
	workDir := setupRepoDir("/tmp", "", "github", "")
	if workDir != "/tmp/github.com" {
		t.Errorf("Expected /tmp/github.com, Got %v", workDir)
	}

	workDir = setupRepoDir("/tmp", "", "github", "https://company.github.com")
	if workDir != "/tmp/company.github.com" {
		t.Errorf("Expected /tmp/company.github.com, Got %v", workDir)
	}

	workDir = setupRepoDir("/tmp", "", "gitlab", "")
	if workDir != "/tmp/gitlab.com" {
		t.Errorf("Expected /tmp/gitlab.com, Got %v", workDir)
	}

	var expectedWorkDir string

	workDir = setupRepoDir("gitlab:///", "", "github", "")
	homeDir, err := homedir.Dir()
	if err == nil {
		expectedWorkDir = path.Join(homeDir, ".gitbackup", "github.com")
	} else {
		log.Fatal("Could not determine home directory")
	}
	if workDir != expectedWorkDir {
		t.Errorf("Expected %v, Got %v", expectedWorkDir, workDir)
	}

	workDir = setupRepoDir("gitlab:///", "/tmp", "gitlab", "")
	expectedWorkDir = "/tmp/gitlab.com"

	if workDir != expectedWorkDir {
		t.Errorf("Expected %v, Got %v", expectedWorkDir, workDir)
	}
}
