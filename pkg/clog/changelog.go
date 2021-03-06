package clog

import (
	"fmt"
	"strings"
	"time"
)

type Changelog struct {
	Header   []string
	Releases []Release
}

func (c *Changelog) AddRelease(version, pullRequestURL, pullRequestBody string) error {
	for _, release := range c.Releases {
		if release.Version == version {
			return fmt.Errorf("version %s already exists in changelog", version)
		}
	}

	location, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return err
	}

	date := time.Now().In(location).Format("2006-01-02")
	release, err := parseRelease(strings.Split(pullRequestBody, "\n"))
	if err != nil {
		return err
	}
	release.Version = version
	release.PullRequestURL = pullRequestURL
	release.Date = date
	c.Releases = append([]Release{release}, c.Releases...)
	return nil
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
	Date           string
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
