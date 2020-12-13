package gitops

import (
	"context"
	"fmt"
	"log"
)

// UpdateFilesParams are parameters for UpdateFiles function.
type UpdateFilesParams struct {
	// Repo is local clone of remote repository.
	Repo repositorier
	// ExportEnv is an environment variable exporter.
	ExportEnv envExporter
	// Renderer renders templates to a given repository.
	Renderer renderAllFileser

	// PullRequest won't push to the branch. It will open a PR only instead.
	PullRequest bool
	// PullRequestTitle is the title of the opened pull request.
	PullRequestTitle string
	// PullRequestBody is the body of the opened pull request.
	PullRequestBody string
	// CommitMessage is the created commit's message.
	CommitMessage string
}

// UpdateFiles updates files in a GitOps repository.
// It either pushes changes to the given branch directly
// or opens a pull request for manual approval.
// URL of the pull request is exported to the
// PR_URL environment variable in the latter case.
func UpdateFiles(ctx context.Context, p UpdateFilesParams) error {
	// Render all templates to the local clone of the repository.
	if err := p.Renderer.renderAllFiles(); err != nil {
		return fmt.Errorf("render all files: %w", err)
	}

	// If rendering the templates didn't cause any changes, we are done here.
	clean, err := p.Repo.workingDirectoryClean()
	if err != nil {
		return fmt.Errorf("checking if working directory is clean: %w", err)
	}
	if clean {
		log.Println("Deployment configuration didn't change, nothing to push.")
		return nil
	}

	if p.PullRequest {
		// Changes are pushed to a new branch in PR-only mode.
		if err := p.Repo.gitCheckoutNewBranch(); err != nil {
			return fmt.Errorf("git push: %w", err)
		}
	}
	// Commit all local changes to the current branch
	// and push them to the remote repository.
	if err := p.Repo.gitCommitAndPush(p.CommitMessage); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	// If we aren't running in PR mode, we are done here
	// (changes were pushed directly to the given branch).
	if !p.PullRequest {
		return nil
	}

	// Open Github pull request.
	prURL, err := p.Repo.openPullRequest(ctx, p.PullRequestTitle, p.PullRequestBody)
	if err != nil {
		return fmt.Errorf("open pull request: %w", err)
	}
	// Export it's URL as an environment variable (following steps can use it).
	if err := p.ExportEnv("PR_URL", prURL); err != nil {
		return fmt.Errorf("export PR_URL env var: %w", err)
	}
	return nil
}
