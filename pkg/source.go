package kraken

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v40/github"
	"golang.org/x/oauth2"

	"github.com/podium-education/kraken/pkg/clog"
)

type Source struct {
	ctx          context.Context
	organization string
	project      string
	workDir      string
	ghClient     *github.Client
	gitAuth      *http.BasicAuth
}

func NewSource(githubToken, githubRepo string) Source {
	ctx := context.Background()
	githubOrg, githubProject := parseRepo(githubRepo)
	dir, err := ioutil.TempDir("/tmp", "kraken-*")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return Source{
		ctx:          ctx,
		organization: githubOrg,
		project:      githubProject,
		workDir:      dir,
		ghClient:     makeGithubClient(ctx, githubToken),
		gitAuth: &http.BasicAuth{
			Username: "me",
			Password: githubToken,
		},
	}
}

func (s Source) GetPullRequest(gitCommit string) *github.PullRequest {
	page := 1
	for page != 0 {
		pullRequests, response, err := s.ghClient.PullRequests.List(
			s.ctx,
			s.organization,
			s.project,
			&github.PullRequestListOptions{
				State: "all",
				ListOptions: github.ListOptions{
					Page: page,
				},
			})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		for _, pullRequest := range pullRequests {
			if pullRequest != nil && pullRequest.MergeCommitSHA != nil && *pullRequest.MergeCommitSHA == gitCommit {
				return pullRequest
			}
		}
		page = response.NextPage
	}
	return nil
}

func (s Source) CloneWiki() error {
	_, err := git.PlainClone(s.workDir, false, &git.CloneOptions{
		URL:  fmt.Sprintf("https://github.com/%s/%s.wiki.git", s.organization, s.project),
		Auth: s.gitAuth,
	})
	return err
}

func (s Source) ParseChangelog() (clog.Changelog, error) {
	out, err := ioutil.ReadFile(filepath.Join(s.workDir, "Changelog.md"))
	if err != nil {
		return clog.Changelog{}, err
	}
	return clog.Parse(string(out))
}

func (s Source) UpdateChangelog(changelog clog.Changelog) (err error) {
	repo, err := git.PlainOpen(s.workDir)
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(s.workDir, "Changelog.md"), []byte(changelog.String()), fs.ModePerm); err != nil {
		return err
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	_, err = worktree.Add(".")
	if err != nil {
		return err
	}

	location, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return err
	}

	_, err = worktree.Commit(fmt.Sprintf("Update from kraken - %s", time.Now().Format("2006-01-02 15:04")), &git.CommitOptions{
		Author: &object.Signature{
			Name:  "kraken",
			Email: "kraken",
			When:  time.Now().In(location),
		},
	})
	if err != nil {
		return err
	}

	return repo.Push(&git.PushOptions{
		Auth: s.gitAuth,
	})
}

func makeGithubClient(ctx context.Context, token string) *github.Client {
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tokenClient := oauth2.NewClient(ctx, tokenSource)
	return github.NewClient(tokenClient)
}

func parseRepo(githubRepo string) (string, string) {
	githubRepoParts := strings.Split(githubRepo, "/")
	return githubRepoParts[0], githubRepoParts[1]
}
