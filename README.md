# gitbackup - Backup your GitHub, GitLab, and Bitbucket repositories
Code Quality [![Go Report Card](https://goreportcard.com/badge/github.com/amitsaha/gitbackup)](https://goreportcard.com/report/github.com/amitsaha/gitbackup)
Linux/Mac OS X [![Build Status](https://travis-ci.org/amitsaha/gitbackup.svg?branch=master)](https://travis-ci.org/amitsaha/gitbackup) Windows [![Build status](https://ci.appveyor.com/api/projects/status/fwki40x1havyian2/branch/master?svg=true)](https://ci.appveyor.com/project/amitsaha/gitbackup/branch/master) 

``gitbackup`` is a tool to backup your git repositories from GitHub (including GitHub enterprise),
GitLab (including custom GitLab installations), or Bitbucket.

``gitbackup`` only creates a backup of the repository and does not currently support issues,
pull requests or other data associated with a git repository. This may or may not be in the future
scope of this tool.

If you are following along my Linux Journal article (published in 2017), please obtain the version of the 
source tagged with [lj-0.1](https://github.com/amitsaha/gitbackup/releases/tag/lj-0.1).

## Installing `gitbackup`

Binary releases are available from the [Releases](https://github.com/amitsaha/gitbackup/releases/) page. Please download the binary corresponding to your OS
and architecture and copy the binary somewhere in your ``$PATH``. It is recommended to rename the binary to `gitbackup` or `gitbackup.exe` (on Windows).

## Using `gitbackup`

``gitbackup`` requires a [GitHub API access token](https://github.com/blog/1509-personal-api-tokens) for
backing up GitHub repositories, a [GitLab personal access token](https://gitlab.com/profile/personal_access_tokens)
for GitLab repositories, and a username and [app password](https://bitbucket.org/account/settings/app-passwords/) for
Bitbucket repositories.

You can supply the tokens to ``gitbackup`` using ``GITHUB_TOKEN`` and ``GITLAB_TOKEN`` environment variables
respectively, and the Bitbucket credentials with ``BITBUCKET_USERNAME`` and ``BITBUCKET_PASSWORD``.

### OAuth Scopes required

#### GitHub

- `repo`: Reading repositories, including private repositories
- `user - read:user`: Reading the authenticated user details. This is only needed for retrieving your username when cloning
via HTTPS and retrieving private repositories.

#### GitLab

- `api`: Grants complete read/write access to the API, including all groups and projects.
For some reason, `read_user` and `read_repository` is not sufficient.

### Security and credentials

When you provide the tokens via environment variables, they remain accessible in your shell history 
and via the processes' environment for the lifetime of the process. By default, SSH authentication
is used to clone your repositories. If `use-https-clone` is specified, private repositories
are cloned via `https` basic auth and the token provided will be stored  in the repositories' 
`.git/config`.

### Examples

Typing ``-help`` will display the command line options that `gitbackup` recognizes:

```
$ gitbackup -help
Usage of ./bin/gitbackup:
  -backupdir string
        Backup directory
  -githost.url string
        DNS of the custom Git host
  -github.repoType string
        Repo types to backup (all, owner, member, starred) (default "all")
  -gitlab.projectMembershipType string
        Project type to clone (all, owner, member) (default "all")
  -gitlab.projectVisibility string
        Visibility level of Projects to clone (internal, public, private) (default "internal")
  -ignore-private
    	Ignore private repositories/projects
  -service string
    	Git Hosted Service Name (github/gitlab/bitbucket)
  -use-https-clone
    	Use HTTPS for cloning instead of SSH
  -bare
    	Clone bare repositories instead of working directories
```
### Backing up your GitHub repositories

To backup all your own GitHub repositories to the default backup directory (``$HOME/.gitbackup/``):

```lang=bash
$ GITHUB_TOKEN=secret$token gitbackup -service github
```

To backup only the GitHub repositories which you are the "owner" of:

```lang=bash
$ GITHUB_TOKEN=secret$token gitbackup -service github -github.repoType owner
```

To backup only the GitHub repositories which you are the "member" of:

```lang=bash
$ GITHUB_TOKEN=secret$token gitbackup -service github -github.repoType member
```

Separately, to backup GitHub repositories you have starred:

```lang=bash
$ GITHUB_TOKEN=secret$token gitbackup -service github -github.repoType starred
```

### Backing up your GitLab repositories

To backup all projects you either own or are a member of which have their [visibility](https://docs.gitlab.com/ce/api/projects.html#project-visibility-level) set to
"internal" on ``https://gitlab.com`` to the default backup directory (``$HOME/.gitbackup/``):

```lang=bash
$ GITLAB_TOKEN=secret$token gitbackup -service gitlab
```

To backup only the GitLab projects (either you are an owner or member of) which are "public"

```lang=bash
$ GITLAB_TOKEN=secret$token gitbackup -service gitlab -gitlab.projectVisibility public
```

To backup only the private repositories (either you are an owner or member of):

```lang=bash
$ GITLAB_TOKEN=secret$token gitbackup -service gitlab -gitlab.projectVisibility private
```

To backup public repositories which you are an owner of:

```lang=bash
$ GITLAB_TOKEN=secret$token gitbackup \
    -service gitlab \
    -gitlab.projectVisibility public \
    -gitlab.projectMembershipType owner
```

To backup public repositories which you are an member of:

```lang=bash
$ GITLAB_TOKEN=secret$token gitbackup \
    -service gitlab \
    -gitlab.projectVisibility public \
    -gitlab.projectMembershipType member
```

### GitHub Enterprise or custom GitLab installation

To specify a custom GitHub enterprise or GitLab location, specify the ``service`` as well as the
the ``githost.url`` flag, like so

```lang=bash
$ GITLAB_TOKEN=secret$token gitbackup -service gitlab -githost.url https://git.yourhost.com
```

### Backing up your Bitbucket repositories

To backup all your Bitbucket repositories to the default backup directory (``$HOME/.gitbackup/``):

```lang=bash
$ BITBUCKET_USERNAME=username BITBUCKET_PASSWORD=password gitbackup -service bitbucket
```

### Specifying a backup location

To specify a custom backup directory, we can use the ``backupdir`` flag:

```lang=bash
$ GITHUB_TOKEN=secret$token gitbackup -service github -backupdir /data/
```

This will create a ``github.com`` directory in ``/data`` and backup all your repositories there instead.
Similarly, it will create a ``gitlab.com`` directory, if you are backing up repositories from ``gitlab``, and a
``bitbucket.com`` directory if you are backing up from Bitbucket.
If you have specified a Git Host URL, it will create a directory structure ``data/host-url/``.


### Cloning bare repositories

To clone bare repositories, we can use the ``bare`` flag:

```lang=bash
$ GITHUB_TOKEN=secret$token gitbackup -service github -bare
```

This will create a directory structure like ``github.com/org/repo.git`` containing bare repositories.


## Building

If you have Golang 1.16.x+ installed, you can clone the repository and:
```
$ go build
```

The built binary will be ``gitbackup``.
