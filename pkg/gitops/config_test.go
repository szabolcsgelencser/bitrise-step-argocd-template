package gitops

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var parseMapCases = map[string]struct {
	s    string
	want map[string]string
}{
	"values don't contain spaces": {
		s: "map[repository:my-repo tag:my-tag]",
		want: map[string]string{
			"repository": "my-repo",
			"tag":        "my-tag",
		},
	},
	"values contain spaces": {
		s: "map[repository:my repo tag:my tag]",
		want: map[string]string{
			"repository": "my repo",
			"tag":        "my tag",
		},
	},
	"values contain multiple spaces": {
		s: "map[repository:my favorite repo so far tag:my little tag extra:some extra variable]",
		want: map[string]string{
			"repository": "my favorite repo so far",
			"tag":        "my little tag",
			"extra":      "some extra variable",
		},
	},
	"there are numbers as well as uppercase characters": {
		s: "map[appVersion:0.2.0 repository:us.gcr.io/my/repo tag:some-other-tag]",
		want: map[string]string{
			"repository": "us.gcr.io/my/repo",
			"tag":        "some-other-tag",
			"appVersion": "0.2.0",
		},
	},
}

func TestParseMap(t *testing.T) {
	for name, tc := range parseMapCases {
		t.Run(name, func(t *testing.T) {
			got := parseMap(tc.s)
			require.Equal(t, tc.want, got)
		})
	}
}
