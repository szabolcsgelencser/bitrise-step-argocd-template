package gitops

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var updateFilesCases = map[string]struct {
	wdClean          bool
	pullRequest      bool
	pullRequestTitle string
	pullRequestBody  string
	pullRequestURL   string
	commitMessage    string
}{
	"no changes to commit": {
		wdClean: true,
	},
	"pushing directly to a branch": {
		commitMessage: "pushing directly to a branch",
	},
	"opening a pull request": {
		pullRequest:      true,
		pullRequestTitle: "my title",
		pullRequestBody:  "my pr body",
		pullRequestURL:   "https://github.com/foo/bar/pr/1",
		commitMessage:    "commit to another branch for a pr",
	},
}

func TestUpdateFiles(t *testing.T) {
	for name, tc := range updateFilesCases {
		t.Run(name, func(t *testing.T) {
			// Mock of local repository.
			var gotNewBranch bool
			var gotCommitMessage string
			var gotPRTitle, gotPRBody string
			repo := &repositorierMock{
				workingDirectoryCleanFunc: func() (bool, error) {
					return tc.wdClean, nil
				},
				gitCheckoutNewBranchFunc: func() error {
					gotNewBranch = true
					return nil
				},
				gitCommitAndPushFunc: func(message string) error {
					gotCommitMessage = message
					return nil
				},
				openPullRequestFunc: func(_ context.Context, title string, body string) (string, error) {
					gotPRTitle = title
					gotPRBody = body
					return tc.pullRequestURL, nil
				},
			}
			// Mock of env exporter function.
			var gotEnvVarName, gotEnvVarValue string
			exportEnv := func(name, value string) error {
				gotEnvVarName = name
				gotEnvVarValue = value
				return nil
			}
			// Mock of templates renderer.
			var gotFilesRendered bool
			renderer := &renderAllFileserMock{
				renderAllFilesFunc: func() error {
					gotFilesRendered = true
					return nil
				},
			}

			ctx := context.Background()
			err := UpdateFiles(ctx, UpdateFilesParams{
				Repo:      repo,
				ExportEnv: exportEnv,
				Renderer:  renderer,

				PullRequest:      tc.pullRequest,
				PullRequestTitle: tc.pullRequestTitle,
				PullRequestBody:  tc.pullRequestBody,
				CommitMessage:    tc.commitMessage,
			})
			require.NoError(t, err, "UpdateFiles")

			assert.True(t, gotFilesRendered, "all files are rendered")
			if tc.wdClean {
				assert.Empty(t, gotCommitMessage, "didn't commit any changes")
				return
			}
			assert.Equal(t, tc.commitMessage, gotCommitMessage, "commit message")
			if !tc.pullRequest {
				assert.False(t, gotNewBranch, "didn't create a new branch")
				return
			}
			assert.True(t, gotNewBranch, "created a new branch")
			assert.Equal(t, "PR_URL", gotEnvVarName, "PR_URL env var name")
			assert.Equal(t, tc.pullRequestURL, gotEnvVarValue, "PR_URL value")
			assert.Equal(t, tc.pullRequestTitle, gotPRTitle, "pr title")
			assert.Equal(t, tc.pullRequestBody, gotPRBody, "pr body")
		})
	}
}
