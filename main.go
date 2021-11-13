package main

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v40/github"
	"github.com/podium-education/kraken/pkg/clog"
	"golang.org/x/oauth2"
)

const (
	GitHubOrg   = "podium-education"
	AuthorName  = "kraken"
	AuthorEmail = "kraken@podiumeducation.com"
)

var (
	auth = &http.BasicAuth{
		Username: "me",
		Password: os.Getenv("GITHUB_TOKEN"),
	}
	tmpDir string
)

func main() {
	_, tokenFound := os.LookupEnv("GITHUB_TOKEN")
	if !tokenFound {
		fmt.Println("GITHUB_TOKEN environment variable not found")
		os.Exit(1)
	}

	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Println("please provide PR number as argument")
		os.Exit(1)
	}
	pullRequestNumber, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var dryRun bool
	if len(args) == 2 && args[1] == "--dry-run" {
		dryRun = true
	}

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	project := path.Base(currentDir)

	fmt.Printf("Running for %s - PR %d | dry run=%v \n", project, pullRequestNumber, dryRun)
	makeWorkspace()
	wikiRepo, wikiPath := getWikiRepo(project)
	changelog := clog.Parse(readChangelog(wikiPath))
	version, pullRequestURL, pullRequestBody := getPullRequestDetails(project, pullRequestNumber)
	fmt.Printf("Attempting to add version %s to changelog...\n", version)
	changelog.AddRelease(version, pullRequestURL, pullRequestBody)
	if writeChangelog(wikiPath, changelog.String()) && !dryRun {
		fmt.Printf("Pushing changes to %s...\n", project)
		commitAndPush(wikiRepo)
	}
	deleteWorkspace()
	fmt.Println("Done!")
}

func makeWorkspace() {
	dir, err := ioutil.TempDir("/tmp", "kraken-*")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	tmpDir = dir
}

func getWikiRepo(project string) (*git.Repository, string) {
	return getRepo(
		getWikiCloneURL("."),
		fmt.Sprintf("%s/%s.wiki", tmpDir, project))
}

func getProjectRepo(project string) (*git.Repository, string) {
	return getRepo(
		getCloneURL("."),
		fmt.Sprintf("%s/%s", tmpDir, project))
}

func getRepo(cloneURL, destination string) (*git.Repository, string) {
	_, err := git.PlainClone(destination, false, &git.CloneOptions{
		URL:  cloneURL,
		Auth: auth,
	})

	if err != nil && err.Error() != "repository already exists" {
		fmt.Println(err)
		os.Exit(1)
	}

	repo, err := git.PlainOpen(destination)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// git reset --hard
	err = worktree.Reset(&git.ResetOptions{
		Mode: git.HardReset,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// git pull origin
	err = worktree.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil && err.Error() != "already up-to-date" {
		fmt.Println(err)
		os.Exit(1)
	}

	return repo, destination
}

func getWikiCloneURL(path string) string {
	return strings.ReplaceAll(getCloneURL(path), ".git", ".wiki.git")
}

func getCloneURL(path string) string {
	repo, err := git.PlainOpen(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	config, err := repo.Config()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	remoteURL := config.Remotes["origin"].URLs[0]
	cloneURL := strings.ReplaceAll(remoteURL, "git@github.com:", "https://github.com/")
	return cloneURL
}

func getPullRequestDetails(project string, pullRequestNumber int) (version string, url string, body string) {
	ctx := context.Background()
	// get pull request body
	githubClient := makeGithubClient(ctx)
	pullRequests, _, err := githubClient.PullRequests.List(
		ctx,
		GitHubOrg,
		project,
		&github.PullRequestListOptions{
			State: "all",
		})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var foundPullRequest *github.PullRequest
	for _, pullRequest := range pullRequests {
		if pullRequest != nil &&
			pullRequest.GetNumber() == pullRequestNumber {
			if pullRequest.GetState() != "open" {
				fmt.Printf("Warning! Pull Request #%d is not open\n", pullRequestNumber)
			}
			foundPullRequest = pullRequest

		}
	}
	if foundPullRequest == nil {
		fmt.Println("no pull request found")
	}

	pullRequestCommit := foundPullRequest.GetHead().GetSHA()
	projectRepo, projectPath := getProjectRepo(project)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	projectWorktree, err := projectRepo.Worktree()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = projectWorktree.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(pullRequestCommit),
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	out, err := ioutil.ReadFile(filepath.Join(projectPath, ".SEMVER"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	version = strings.TrimSpace(string(out))

	var pullRequestBody string
	if foundPullRequest.GetUser().GetLogin() == "dependabot[bot]" {
		pullRequestBody =
			`
### Security
- Dependabot bumped dependencies
`
	} else {
		pullRequestBody = foundPullRequest.GetBody()
	}
	return version, foundPullRequest.GetURL(), pullRequestBody
}

func makeGithubClient(ctx context.Context) *github.Client {
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tokenClient := oauth2.NewClient(ctx, tokenSource)
	return github.NewClient(tokenClient)
}

func readChangelog(wikiPath string) string {
	out, err := ioutil.ReadFile(filepath.Join(wikiPath, "Changelog.md"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return string(out)
}

func writeChangelog(wikiPath, changelog string) bool {
	existingChangelog := readChangelog(wikiPath)
	if cmp.Equal(existingChangelog, changelog) {
		return false
	}

	err := ioutil.WriteFile(filepath.Join(wikiPath, "Changelog.md"), []byte(changelog), fs.ModePerm)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return true
}

func commitAndPush(repo *git.Repository) {
	worktree, err := repo.Worktree()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = worktree.Add(".")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = worktree.Commit(fmt.Sprintf("Update from kraken - %s", time.Now().Format("2006-01-02 15:04")), &git.CommitOptions{
		Author: &object.Signature{
			Name:  AuthorName,
			Email: AuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = repo.Push(&git.PushOptions{
		Auth: auth,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func deleteWorkspace() {
	err := os.RemoveAll(tmpDir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
