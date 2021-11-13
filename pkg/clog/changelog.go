package clog

import (
	"fmt"
	"strings"

	"github.com/podium-education/etcetera/when"
)

type Changelog struct {
	Header   []string
	Releases []Release
}

func (c *Changelog) AddRelease(version, pullRequestURL, pullRequestBody string) {
	date, _ := when.Now()
	release := parseRelease(strings.Split(pullRequestBody, "\n"))
	release.Version = version
	release.PullRequestURL = pullRequestURL
	release.Date = date
	c.Releases = append([]Release{release}, c.Releases...)
}

func (c Changelog) String() string {
	var sb strings.Builder
	sb.WriteString(strings.Join(c.Header, "\n"))
	sb.WriteString("\n")
	var releases []string
	for _, release := range c.Releases {
		releases = append(releases, release.String())
	}
	sb.WriteString(strings.Join(releases, "\n\n"))
	return sb.String()
}

type Release struct {
	Version        string
	PullRequestURL string
	Date           when.Date
	Changes        []Change
}

func (r Release) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## [%s](%s) - %s\n", r.Version, r.PullRequestURL, r.Date))
	var changes []string
	for _, change := range r.Changes {
		changes = append(changes, change.String())
	}
	sb.WriteString(strings.Join(changes, "\n\n"))
	return sb.String()
}

type Change struct {
	Kind    string
	Entries []string
}

func (c Change) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### %s\n", c.Kind))
	var entries []string
	for _, entry := range c.Entries {
		entries = append(entries, fmt.Sprintf("- %s", entry))
	}
	sb.WriteString(strings.Join(entries, "\n"))
	return sb.String()
}
