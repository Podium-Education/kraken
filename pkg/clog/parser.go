package clog

import (
	"regexp"
	"strings"

	"github.com/podium-education/etcetera/cheat"
	"github.com/podium-education/etcetera/when"
)

func Parse(raw string) (changelog Changelog) {
	changelogLines := strings.Split(raw, "\n")
	header, headerEndIndex := extractHeader(changelogLines)
	changelog.Header = header
	if headerEndIndex == 0 {
		// no releases found in changelog
		return
	}

	rawReleases := extractReleases(changelogLines[headerEndIndex:])
	var releases []Release
	for _, rawRelease := range rawReleases {
		releases = append(releases, parseRelease(rawRelease))
	}
	changelog.Releases = releases
	return
}

func extractHeader(changelogLines []string) ([]string, int) {
	var headerEndIndex int
	var header []string
	for i, changelogLine := range changelogLines {
		if strings.HasPrefix(strings.TrimSpace(changelogLine), "## ") {
			headerEndIndex = i
			break
		}
		header = append(header, changelogLine)
	}
	return header, headerEndIndex
}

func extractReleases(releaseLines []string) [][]string {
	var releases [][]string
	var release []string
	for i, releaseLine := range releaseLines {
		if strings.HasPrefix(strings.TrimSpace(releaseLine), "## ") && i != 0 {
			releases = append(releases, release)
			release = []string{} // reset
		}
		release = append(release, releaseLine)
		// add the last release
		if i == len(releaseLines)-1 {
			releases = append(releases, release)
		}
	}
	return releases
}

func parseRelease(releaseLines []string) Release {
	var release Release
	var change Change
	for i, releaseLine := range releaseLines {
		if i == 0 && strings.HasPrefix(releaseLine, "## ") {
			version, pullRequestURL, date := parseReleaseHeader(releaseLine)
			release = Release{
				Version:        version,
				PullRequestURL: pullRequestURL,
				Date:           date,
			}
			continue
		}

		if strings.HasPrefix(releaseLine, "### ") {
			if change.Kind != "" {
				release.Changes = append(release.Changes, change)
				change = Change{}
			}
			kind := parseChangeHeader(releaseLine)
			change.Kind = kind
			continue
		}

		if strings.TrimSpace(releaseLine) != "" {
			change.Entries = append(change.Entries, cleanChangeEntry(releaseLine))
		}

		// add last change to release
		if i == len(releaseLines)-1 {
			release.Changes = append(release.Changes, change)
		}
	}
	return release
}

func parseReleaseHeader(rawHeader string) (string, string, when.Date) {
	pattern := regexp.MustCompile(`##\s\[(\d+\.\d+\.\d+)]\((https:[/a-zA-Z0-9.-]+)\)\s+-\s+(\d{4}-\d{1,2}-\d{1,2})`)
	matches := pattern.FindAllStringSubmatch(rawHeader, -1)
	return matches[0][1], matches[0][2], cheat.Date(matches[0][3])
}

func parseChangeHeader(rawHeader string) string {
	pattern := regexp.MustCompile(`###\s(.*)`)
	matches := pattern.FindAllStringSubmatch(rawHeader, -1)
	return matches[0][1]
}

func cleanChangeEntry(rawEntry string) string {
	rawEntry = strings.TrimSpace(rawEntry)
	rawEntry = strings.TrimPrefix(rawEntry, "-")
	rawEntry = strings.TrimSpace(rawEntry)
	return rawEntry
}
