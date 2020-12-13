package gitops

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var githubOwnerRepoCases = map[string]struct {
	s                   string
	wantOwner, wantRepo string
	wantErr             bool
}{
	"simple ssh url for github": {
		s:         "git@github.com:szabolcsgelencser/sample-deploy-config.git",
		wantOwner: "szabolcsgelencser",
		wantRepo:  "sample-deploy-config",
	},
	"another simple ssh url for github": {
		s:         "git@github.com:bitrise-io/den.git",
		wantOwner: "bitrise-io",
		wantRepo:  "den",
	},
	"unsupported https url for github": {
		s:       "https://github.com/bitrise-io/den.git",
		wantErr: true,
	},
	"malformed ssh url (missing prefix)": {
		s:       "bitrise-io/den.git",
		wantErr: true,
	},
	"malformed ssh url (missing postfix)": {
		s:       "git@github.com:bitrise-io/den",
		wantErr: true,
	},
	"malformed ssh url (not having owner/repo)": {
		s:       "git@github.com:den.git",
		wantErr: true,
	},
}

func TestGithubOwnerRepo(t *testing.T) {
	for name, tc := range githubOwnerRepoCases {
		t.Run(name, func(t *testing.T) {
			gotOwner, gotRepo, gotErr := githubOwnerRepo(tc.s)
			if tc.wantErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			require.Equal(t, tc.wantOwner, gotOwner)
			require.Equal(t, tc.wantRepo, gotRepo)
		})
	}
}
