package gitops

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var repositoryCases = map[string]struct {
	upstreamBranch string
	repoURL        string
}{
	"master of testhub.assert/test/foo": {
		upstreamBranch: "master",
		repoURL:        "testhub.assert/test/foo",
	},
	"staging of testhub.assert/test/bar": {
		upstreamBranch: "staging",
		repoURL:        "testhub.assert/test/bar",
	},
}

func TestRepository(t *testing.T) {
	ctx := context.Background()

	for name, tc := range repositoryCases {
		t.Run(name, func(t *testing.T) {
			upstreamPath, close := localUpstreamRepo(t, tc.upstreamBranch)
			defer close()

			// Initialize mock Github client.
			wantPullRequestURL := fmt.Sprintf("https://%s/pr/15", tc.repoURL)
			var gotHead, gotBase string
			gh := &githuberMock{
				OpenPullRequestFunc: func(_ context.Context, p openPullRequestParams) (string, error) {
					gotHead = p.head
					gotBase = p.base
					return wantPullRequestURL, nil
				},
			}

			// Initialize mock SSH key.
			var gotKeyClosed bool
			sshKey := &sshKeyerMock{
				privateKeyPathFunc: func() string {
					return ""
				},
				closeFunc: func(ctx context.Context) []error {
					gotKeyClosed = true
					return nil
				},
			}

			// Create new local repository clone.
			repo, err := NewRepository(ctx, NewRepositoryParams{
				Github: gh,
				SSHKey: sshKey,
				Remote: RemoteConfig{
					URL:    upstreamPath,
					Branch: tc.upstreamBranch,
				},
			})
			require.NoError(t, err, "newRepository")

			// The repository is clean if there weren't any changes.
			clean, err := repo.workingDirectoryClean()
			require.True(t, clean, "working directory is clean without changes")

			// It's dirty after making some changes.
			changePath := path.Join(repo.localPath(), "empty.go")
			write(t, changePath, "package empty")

			clean, err = repo.workingDirectoryClean()
			require.False(t, clean, "working directory is dirty after changes")

			// Commit and push changes to upstream repository.
			err = repo.gitCommitAndPush("test commit")
			require.NoError(t, err, "commit and push test")

			clean, err = repo.workingDirectoryClean()
			require.True(t, clean, "working directory is clean after commit")

			// Can create a new branch and push it to upstream as well,
			// open new pull request from it to the base branch.
			require.NoError(t, repo.gitCheckoutNewBranch(), "new branch")
			changePath = path.Join(repo.localPath(), "another.go")
			write(t, changePath, "package another")

			err = repo.gitCommitAndPush("another commit")
			require.NoError(t, err, "commit and push another")

			gotPullRequestURL, err := repo.openPullRequest(ctx, "", "")
			require.NoError(t, err, "open pull request")
			assert.Equal(t, wantPullRequestURL, gotPullRequestURL, "pr url")

			assert.Equal(t, tc.upstreamBranch, gotBase, "pr base")

			assert.NotEqual(t, gotBase, gotHead, "pr head differs from base")
			wantHead, err := repo.currentBranch()
			require.NoError(t, err, "current branch")
			assert.Equal(t, wantHead, gotHead, "pr head = current branch")

			// Assert propagation of close to ssh key as well.
			require.False(t, gotKeyClosed, "key wasn't closed before")
			require.Nil(t, repo.Close(ctx), "repo.Close")
			require.True(t, gotKeyClosed, "key is closed by repo")
		})
	}
}

func localUpstreamRepo(t *testing.T, branch string) (string, func()) {
	repoPath, err := ioutil.TempDir("", "")
	require.NoError(t, err, "new temp directory for local upstream")
	readmePath := path.Join(repoPath, "README.md")

	git(t, repoPath, "init", "-b", branch)
	write(t, readmePath, "A local upstream repository for testing.")
	git(t, repoPath, "add", "--all")
	git(t, repoPath, "commit", "-m", "initial commit")
	// allow push from another git repository
	git(t, repoPath, "config", "receive.denyCurrentBranch", "ignore")

	return repoPath, func() {
		os.RemoveAll(repoPath)
	}
}

func write(t *testing.T, path, content string) {
	err := ioutil.WriteFile(path, []byte(content), 0600)
	require.NoError(t, err, "ioutil.WriteFile(%s)", path)
}

func git(t *testing.T, repoPath string, args ...string) {
	// Change current directory to the repositorys local clone.
	originalDir, err := os.Getwd()
	require.NoError(t, err, "get current dir")
	require.NoError(t, os.Chdir(repoPath), "change to upstream repo")
	// Defer a revert of the current directory to the original one.
	defer func() {
		require.NoError(t, os.Chdir(originalDir), "change to original dir")
	}()

	cmd := exec.Command("git", args...)
	require.NoError(t, cmd.Run(), "git %+v", args)
}
