package main

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
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
	log.SetFlags(log.Lshortfile)

	_, tokenFound := os.LookupEnv("GITHUB_TOKEN")
	if !tokenFound {
		log.Fatalln("GITHUB_TOKEN environment variable not found")
	}

	args := os.Args[1:]
	if len(args) != 1 {
		log.Fatalln("please provide PR number as argument")
	}
	pullRequestNumber, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatalln(err)
	}
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}
	project := path.Base(currentDir)

	fmt.Printf("Running for %s - PR %d\n", project, pullRequestNumber)
	makeWorkspace()
	wikiRepo, wikiPath := getWikiRepo(project)
	changelog := clog.Parse(readChangelog(wikiPath))
	version, pullRequestURL, pullRequestBody := getPullRequestDetails(project, pullRequestNumber)
	fmt.Printf("Adding %s to changelog\n", version)
	changelog.AddRelease(version, pullRequestURL, pullRequestBody)
	if writeChangelog(wikiPath, changelog.String()) {
		fmt.Printf("Pushing changes to %s\n", project)
		commitAndPush(wikiRepo)
	}
	deleteWorkspace()

}

func makeWorkspace() {
	dir, err := ioutil.TempDir("/tmp", "kraken-*")
	if err != nil {
		log.Fatalln(err)
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
		URL:      cloneURL,
		Auth:     auth,
		Progress: os.Stdout,
	})

	if err != nil && err.Error() != "repository already exists" {
		log.Fatalln(err)
	}

	repo, err := git.PlainOpen(destination)
	if err != nil {
		log.Fatalln(err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		log.Fatalln(err)
	}

	// git reset --hard
	err = worktree.Reset(&git.ResetOptions{
		Mode: git.HardReset,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// git pull origin
	err = worktree.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil && err.Error() != "already up-to-date" {
		log.Fatalln(err)
	}

	return repo, destination
}

func getWikiCloneURL(path string) string {
	return strings.ReplaceAll(getCloneURL(path), ".git", ".wiki.git")
}

func getCloneURL(path string) string {
	repo, err := git.PlainOpen(path)
	if err != nil {
		log.Fatalln(err)
	}

	config, err := repo.Config()
	if err != nil {
		log.Fatalln(err)
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
		&github.PullRequestListOptions{})
	if err != nil {
		log.Fatalln(err)
	}

	var foundPullRequest *github.PullRequest
	for _, pullRequest := range pullRequests {
		if pullRequest != nil &&
			pullRequest.GetNumber() == pullRequestNumber &&
			pullRequest.GetState() == "open" {
			foundPullRequest = pullRequest
		}
	}
	if foundPullRequest == nil {
		log.Fatalln("no pull request found")
	}

	pullRequestCommit := foundPullRequest.GetHead().GetSHA()
	projectRepo, projectPath := getProjectRepo(project)
	if err != nil {
		log.Fatalln(err)
	}

	projectWorktree, err := projectRepo.Worktree()
	if err != nil {
		log.Fatalln(err)
	}

	err = projectWorktree.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(pullRequestCommit),
	})
	if err != nil {
		log.Fatalln(err)
	}

	out, err := ioutil.ReadFile(filepath.Join(projectPath, ".SEMVER"))
	if err != nil {
		log.Fatalln(err)
	}
	version = strings.TrimSpace(string(out))
	return version, foundPullRequest.GetURL(), foundPullRequest.GetBody()
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
		log.Fatalln(err)
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
		log.Fatalln(err)
	}
	return true
}

func commitAndPush(repo *git.Repository) {
	worktree, err := repo.Worktree()
	if err != nil {
		log.Fatalln(err)
	}

	_, err = worktree.Add(".")
	if err != nil {
		log.Fatalln(err)
	}

	_, err = worktree.Commit(fmt.Sprintf("Update from kraken - %s", time.Now().Format("2006-01-02 15:04")), &git.CommitOptions{
		Author: &object.Signature{
			Name:  AuthorName,
			Email: AuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = repo.Push(&git.PushOptions{
		Auth: auth,
	})
	if err != nil {
		log.Fatalln(err)
	}
}

func deleteWorkspace() {
	err := os.RemoveAll(tmpDir)
	if err != nil {
		log.Fatalln(err)
	}
}
