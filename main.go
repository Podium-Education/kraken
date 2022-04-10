package main

import (
	"flag"
	"fmt"
	"os"

	kraken "github.com/podium-education/kraken/pkg"
)

var (
	githubTokenFlag string
	githubRepoFlag  string
	gitCommitFlag   string
	versionFlag     string
)

func main() {
	src := kraken.NewSource(githubTokenFlag, githubRepoFlag)

	pullRequest := src.GetPullRequest(gitCommitFlag)
	if pullRequest != nil && *pullRequest.Base.Repo.HasWiki == true {
		if err := src.CloneWiki(); err != nil {
			handleError(err)
		}

		changelog, err := src.ParseChangelog()
		if err != nil {
			handleError(err)
		}

		if err = changelog.AddRelease(versionFlag, pullRequest.GetHTMLURL(), pullRequest.GetBody()); err != nil {
			handleError(err)
		}

		if err = src.UpdateChangelog(changelog); err != nil {
			handleError(err)
		}
		_, _ = fmt.Fprintln(os.Stdout, "Added release to Changelog wiki")
		_, _ = fmt.Fprintln(os.Stdout, changelog.Releases[0].String())
	}
}

func init() {
	flag.StringVar(&githubTokenFlag, "github-token", "", "GitHub Access Token for accessing the wiki repo")
	flag.StringVar(&githubRepoFlag, "github-repo", "", "GitHub repository")
	flag.StringVar(&gitCommitFlag, "git-commit", "", "git commit hash")
	flag.StringVar(&versionFlag, "version", "", "The semantic version")
	flag.Parse()
}

func handleError(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "error %s\n", err)
	os.Exit(1)
}
