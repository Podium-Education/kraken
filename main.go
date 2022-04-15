package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	kraken "github.com/podium-education/kraken/pkg"
)

var (
	githubTokenFlag  string
	githubRepoFlag   string
	gitCommitFlag    string
	versionFlag      string
	gitTagModeFlag   string
	gitTagFormatFlag string
)

func main() {
	src := kraken.NewSource(githubTokenFlag, githubRepoFlag)

	pullRequest := src.GetPullRequest(gitCommitFlag)
	if pullRequest != nil {
		gitTagMessage := fmt.Sprintf("kraken created tag for version: %s", versionFlag)
		if *pullRequest.Base.Repo.HasWiki == true {
			if err := src.CloneWiki(); err != nil {
				handleError(err)
			}

			changelog, err := src.ParseChangelog()
			if err != nil {
				handleError(err)
			}

			var pullRequestBody string
			if pullRequest.GetUser().GetLogin() == "dependabot[bot]" {
				pullRequestBody = `### Security
- Dependabot bumped dependencies
`
			} else {
				pullRequestBody = pullRequest.GetBody()
			}

			if err = changelog.AddRelease(versionFlag, pullRequest.GetHTMLURL(), pullRequestBody); err != nil {
				handleError(err)
			}

			if err = src.UpdateChangelog(changelog); err != nil {
				handleError(err)
			}
			_, _ = fmt.Fprintln(os.Stdout, "Added release to Changelog wiki")
			_, _ = fmt.Fprintln(os.Stdout, changelog.Releases[0].String())
		}

		if strings.ToLower(gitTagModeFlag) == "on" {
			if err := src.CloneSource(); err != nil {
				handleError(err)
			}
			if err := src.Tag(gitCommitFlag, versionFlag, gitTagFormatFlag, gitTagMessage); err != nil {
				handleError(err)
			}
			_, _ = fmt.Fprintf(os.Stdout, "Tag created for version: %s", versionFlag)
		}

	}
}

func init() {
	flag.StringVar(&githubTokenFlag, "github-token", "", "GitHub Access Token for accessing the wiki repo")
	flag.StringVar(&githubRepoFlag, "github-repo", "", "GitHub repository")
	flag.StringVar(&gitCommitFlag, "git-commit", "", "git commit hash")
	flag.StringVar(&versionFlag, "version", "", "The semantic version")
	flag.StringVar(&gitTagModeFlag, "git-tag-mode", "off", "Handling of the version git tag")
	flag.StringVar(&gitTagFormatFlag, "git-tag-format", "{version}", "The format of the version git tag")
	flag.Parse()
}

func handleError(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "error %s\n", err)
	os.Exit(1)
}
