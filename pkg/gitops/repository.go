package gitops

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

//go:generate moq -out repository_moq_test.go . repositorier
type repositorier interface {
	Close(ctx context.Context) []error
	localPath() string
	gitClone() error
	workingDirectoryClean() (bool, error)
	gitCheckoutNewBranch() error
	gitCommitAndPush(message string) error
	openPullRequest(ctx context.Context, title, body string) (string, error)
}

// repository implements the repositorier interface.
var _ repositorier = (*repository)(nil)

type repository struct {
	gh          githuber
	remote      RemoteConfig
	sshKey      sshKeyer
	tmpRepoPath string
}

// RemoteConfig is a git remote configuration.
type RemoteConfig struct {
	URL, Branch, Folder string
}

// NewRepositoryParams are parameters for NewRepository function.
type NewRepositoryParams struct {
	Github githuber
	SSHKey sshKeyer
	Remote RemoteConfig
}

// NewRepository returns a new local clone of a remote repository.
// It must be closed after usage and it will also close the SSH key it uses.
func NewRepository(ctx context.Context, p NewRepositoryParams) (*repository, error) {
	// Temporary directory for local clone of repository.
	tmpRepoPath, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, fmt.Errorf("create temp dir for repo: %w", err)
	}
	repo := &repository{
		gh:          p.Github,
		remote:      p.Remote,
		sshKey:      p.SSHKey,
		tmpRepoPath: tmpRepoPath,
	}
	if err := repo.gitClone(); err != nil {
		return nil, fmt.Errorf("git clone repo: %w", err)
	}
	return repo, nil
}

// Close closes all related resoruces of the repository.
func (r repository) Close(ctx context.Context) []error {
	var errs []error
	// Close all resources of temporary deploy key.
	if keyErrs := r.sshKey.close(ctx); keyErrs != nil {
		errs = append(errs, keyErrs...)
	}
	// Delete temporary repository from the local filesystem.
	if err := os.RemoveAll(r.tmpRepoPath); err != nil {
		errs = append(errs, fmt.Errorf("remove temporary repository: %w", err))
	}
	return errs
}

func (r repository) localPath() string {
	return r.tmpRepoPath
}

func (r repository) gitClone() error {
	_, err := r.git("clone",
		"--branch", r.remote.Branch, "--single-branch", r.remote.URL, ".")
	return err
}

func (r repository) workingDirectoryClean() (bool, error) {
	status, err := r.git("status")
	if err != nil {
		return false, err
	}
	return strings.Contains(status, "nothing to commit, working tree clean"), nil
}

func (r repository) gitCheckoutNewBranch() error {
	// Generate branch name based on the current time.
	t := time.Now()
	branch := fmt.Sprintf("ci-%d-%02d-%02dT%02d-%02d-%02d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	// Execute git checkout to a new branch with that name.
	if _, err := r.git("checkout", "-b", branch); err != nil {
		return fmt.Errorf("checkout new branch %q: %w", branch, err)
	}
	return nil
}

func (r repository) gitCommitAndPush(message string) error {
	// Stage all changes, commit them to the current branch
	// and push the commit to the remote repository.
	gitArgs := [][]string{
		{"add", "--all"},
		{"commit", "-m", message},
		{"push", "--all", "-u"},
	}
	for _, a := range gitArgs {
		if _, err := r.git(a...); err != nil {
			return err
		}
	}
	return nil
}

func (r repository) currentBranch() (string, error) {
	branch, err := r.git("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(branch), nil
}

func (r repository) git(args ...string) (string, error) {
	// Change current directory to the repositorys local clone.
	originalDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current dir: %w", err)
	}
	if err := os.Chdir(r.tmpRepoPath); err != nil {
		return "", fmt.Errorf("change dir to %q: %w", r.tmpRepoPath, err)
	}
	// Defer a revert of the current directory to the original one.
	defer os.Chdir(originalDir)

	cmd := exec.Command("git", args...)

	// Specify SSH key for git commands via environment variable.
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf(
		"GIT_SSH_COMMAND=ssh -i %s -o IdentitiesOnly=yes",
		r.sshKey.privateKeyPath(),
	))

	// Run git command and returns it's combined output of stdout and stderr.
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("run command %v: %w (output: %s)", args, err, out)
	}
	return string(out), nil
}

func (r repository) openPullRequest(ctx context.Context, title, body string) (string, error) {
	// PR will be open from the current branch.
	currBranch, err := r.currentBranch()
	if err != nil {
		return "", fmt.Errorf("current branch: %w", err)
	}
	// Open pull request from current branch to the base branch.
	url, err := r.gh.OpenPullRequest(ctx, openPullRequestParams{
		title: title,
		body:  body,
		head:  currBranch,
		base:  r.remote.Branch,
	})
	if err != nil {
		return "", fmt.Errorf("call github: %w", err)
	}
	return url, nil
}
