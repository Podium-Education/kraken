package clog

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestChangelog_String(t *testing.T) {
	testCases := []struct {
		subject Changelog
		want    string
	}{
		{
			Changelog{},
			"\n",
		},
		{
			Changelog{
				Header: []string{"This is the header", "", "I contain important information!"},
			},
			"This is the header\n\nI contain important information!\n",
		},
		{
			Changelog{
				Header: []string{"This is the header", ""},
				Releases: []Release{
					{
						Version:        "0.0.1",
						PullRequestURL: "https://hello.com/pull/1",
						Date:           "2021-10-10",
						Changes: []Change{
							{
								Kind: "Added",
								Entries: []string{
									"Added a new thing",
								},
							},
						},
					},
				},
			},
			"This is the header\n\n## [0.0.1](https://hello.com/pull/1) - 2021-10-10\n### Added\n- Added a new thing",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			got := tc.subject.String()
			if !cmp.Equal(got, tc.want) {
				t.Error(cmp.Diff(got, tc.want))
			}
		})
	}
}

func TestRelease_String(t *testing.T) {
	testCases := []struct {
		subject Release
		want    string
	}{
		{
			Release{},
			"## []() - \n",
		},
		{
			Release{
				Version:        "0.0.1",
				PullRequestURL: "https://hello.com/pull/1",
				Date:           "2021-10-10",
				Changes: []Change{
					{
						Kind: "Added",
						Entries: []string{
							"Added a new thing",
						},
					},
				},
			},
			"## [0.0.1](https://hello.com/pull/1) - 2021-10-10\n### Added\n- Added a new thing",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			got := tc.subject.String()
			if !cmp.Equal(got, tc.want) {
				t.Error(cmp.Diff(got, tc.want))
			}
		})
	}
}

func TestChange_String(t *testing.T) {
	testCases := []struct {
		subject Change
		want    string
	}{
		{
			Change{},
			"### \n",
		},
		{
			Change{
				Kind: "Added",
				Entries: []string{
					"Added a new thing",
				},
			},
			"### Added\n- Added a new thing",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			got := tc.subject.String()
			if !cmp.Equal(got, tc.want) {
				t.Error(cmp.Diff(got, tc.want))
			}
		})
	}
}
