package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/szabolcsgelencser/bitrise-step-argocd-template/pkg/gitops"
)

func main() {
	if err := run(); err != nil {
		log.Printf("error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// Read gitops related config from environment.
	cfg, err := gitops.NewConfig()
	if err != nil {
		return fmt.Errorf("new gitops config: %w", err)
	}

	// Create Github client.
	gh, err := gitops.NewGithub(ctx, cfg.DeployRepositoryURL, cfg.DeployPAT)
	if err != nil {
		return fmt.Errorf("new github client: %w", err)
	}

	// Temporary SSH key (used by git commands).
	sshKey, err := gitops.NewSSHKey(ctx, gh)
	if err != nil {
		return fmt.Errorf("new temporary ssh key: %w", err)
	}

	// Create local clone of the remote repository.
	repo, err := gitops.NewRepository(ctx, gitops.NewRepositoryParams{
		Github: gh,
		SSHKey: sshKey,
		Remote: gitops.RemoteConfig{
			URL:    cfg.DeployRepositoryURL,
			Branch: cfg.DeployBranch,
		},
	})
	defer func() {
		if errs := repo.Close(ctx); errs != nil {
			for _, err := range errs {
				log.Printf("warning: close repo resource: %s\n", err)
			}
		}
	}()
	if err != nil {
		return fmt.Errorf("new repository: %w", err)
	}

	// Create templates renderer.
	renderer := gitops.TemplatesRenderer{
		SourceFolder:      cfg.TemplatesFolder,
		Vars:              cfg.Vars,
		DestinationRepo:   repo,
		DestinationFolder: cfg.DeployFolder,
	}

	// Update files of gitops repository.
	if err := gitops.UpdateFiles(ctx, gitops.UpdateFilesParams{
		Repo:             repo,
		ExportEnv:        gitops.EnvmanExport,
		Renderer:         renderer,
		PullRequest:      cfg.PullRequest,
		PullRequestTitle: cfg.PullRequestTitle,
		PullRequestBody:  cfg.PullRequestBody,
		CommitMessage:    cfg.CommitMessage,
	}); err != nil {
		return fmt.Errorf("update files in gitops repo: %w", err)
	}
	return nil
}
