package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"sync"
	"testing"

	"github.com/spf13/afero"
)

func fakePullCommand(command string, args ...string) (cmd *exec.Cmd) {
	cs := []string{"-test.run=TestHelperPullProcess", "--", command}
	cs = append(cs, args...)
	cmd = exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func fakeCloneCommand(command string, args ...string) (cmd *exec.Cmd) {
	cs := []string{"-test.run=TestHelperCloneProcess", "--", command}
	cs = append(cs, args...)
	cmd = exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func TestBackup(t *testing.T) {
	var wg sync.WaitGroup

	repo := Repository{Name: "testrepo", CloneURL: "git://foo.com/foo"}
	backupDir := "/tmp/backupdir"

	// Memory FS
	appFS = afero.NewMemMapFs()
	appFS.MkdirAll(backupDir, 0771)

	defer func() {
		execCommand = exec.Command
		wg.Wait()
	}()

	// Test clone
	execCommand = fakeCloneCommand
	wg.Add(1)
	stdoutStderr, err := backUp(backupDir, &repo, &wg)
	if err != nil {
		t.Errorf("%s", stdoutStderr)
	}

	// Test pull
	repoDir := path.Join(backupDir, repo.Name)
	appFS.MkdirAll(repoDir, 0771)
	execCommand = fakePullCommand
	wg.Add(1)
	stdoutStderr, err = backUp(backupDir, &repo, &wg)
	if err != nil {
		t.Errorf("%s", stdoutStderr)
	}

}

func TestHelperPullProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Check that git command was executed
	if os.Args[3] != "git" || os.Args[6] != "pull" {
		fmt.Fprintf(os.Stdout, "Expected git pull to be executed. Got %v", os.Args[3:])
		os.Exit(1)
	}
	os.Exit(0)
}

func TestHelperCloneProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Check that git command was executed
	if os.Args[3] != "git" || os.Args[4] != "clone" {
		fmt.Fprintf(os.Stdout, "Expected git clone to be executed. Got %v", os.Args[3:])
		os.Exit(1)
	}
	os.Exit(0)
}

func TestSetupBackupDir(t *testing.T) {
	appFS = afero.NewMemMapFs()
	backupdir := setupBackupDir("/tmp", "github", "")
	if backupdir != "/tmp/github.com" {
		t.Errorf("Expected /tmp/github.com, Got %v", backupdir)
	}

	backupdir = setupBackupDir("/tmp", "github", "https://company.github.com")
	if backupdir != "/tmp/company.github.com" {
		t.Errorf("Expected /tmp/company.github.com, Got %v", backupdir)
	}

	backupdir = setupBackupDir("/tmp", "gitlab", "")
	if backupdir != "/tmp/gitlab.com" {
		t.Errorf("Expected /tmp/gitlab.com, Got %v", backupdir)
	}
}
