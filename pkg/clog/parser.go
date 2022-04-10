package clog

import (
	"errors"
	"regexp"
	"strings"
)

func Parse(raw string) (Changelog, error) {
	changelogLines := strings.Split(raw, "\n")
	header, headerEndIndex := extractHeader(changelogLines)

	var changelog Changelog
	changelog.Header = header
	if headerEndIndex == 0 {
		return changelog, nil
	}

	rawReleases := extractReleases(changelogLines[headerEndIndex:])
	var releases []Release
	for _, rawRelease := range rawReleases {
		release, err := parseRelease(rawRelease)
		if err != nil {
			return changelog, err
		}
		releases = append(releases, release)
	}
	changelog.Releases = releases
	return changelog, nil
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

func parseRelease(releaseLines []string) (Release, error) {
	var release Release
	var change Change

	if !strings.HasPrefix(releaseLines[0], "## ") {
		return Release{}, errors.New("unexpected format for the release notes")
	}

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
	return release, nil
}

func parseReleaseHeader(rawHeader string) (string, string, string) {
	pattern := regexp.MustCompile(`##\s\[(\d+\.\d+\.\d+)]\((https:[/a-zA-Z0-9.-]+)\)\s+-\s+(\d{4}-\d{1,2}-\d{1,2})`)
	matches := pattern.FindAllStringSubmatch(rawHeader, -1)
	return matches[0][1], matches[0][2], matches[0][3]
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
