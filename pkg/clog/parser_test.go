package clog

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/podium-education/etcetera/when"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		testTitle string
		raw       string
		want      Changelog
	}{
		{
			"header only",
			`This is the header

It has information about stuff!`,
			Changelog{
				Header: []string{
					"This is the header",
					"",
					"It has information about stuff!",
				},
			},
		},
		{
			"header and versions",
			`This is the header
## [0.1.2](https://github.com/Hello/stuff-sdk/pull/2) - 2010-10-11
### Changed
- Updated the stuff with more things

### Fixed
- Fixed an issue that was happening
- Fixed a different issue that was happening

## [0.0.1](https://github.com/Hello/stuff-sdk/pull/1) - 2010-10-10
### Added
- Initial commit
`,
			Changelog{
				Header: []string{"This is the header"},
				Releases: []Release{
					{
						Version:        "0.1.2",
						PullRequestURL: "https://github.com/Hello/stuff-sdk/pull/2",
						Date:           when.NewDate(2010, 10, 11),
						Changes: []Change{
							{
								Kind: "Changed",
								Entries: []string{
									"Updated the stuff with more things",
								},
							},
							{
								Kind: "Fixed",
								Entries: []string{
									"Fixed an issue that was happening",
									"Fixed a different issue that was happening",
								},
							},
						},
					},
					{
						Version:        "0.0.1",
						PullRequestURL: "https://github.com/Hello/stuff-sdk/pull/1",
						Date:           when.NewDate(2010, 10, 10),
						Changes: []Change{
							{
								Kind: "Added",
								Entries: []string{
									"Initial commit",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testTitle, func(t *testing.T) {
			got := Parse(tc.raw)
			if !cmp.Equal(got, tc.want) {
				t.Error(cmp.Diff(got, tc.want))
			}
		})
	}
}
