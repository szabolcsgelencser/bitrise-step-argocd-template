package gitops

import (
	"context"
	"fmt"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
	gogh "github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

//go:generate moq -out github_moq_test.go . githuber
type githuber interface {
	AddKey(context.Context, []byte) (int64, error)
	DeleteKey(context.Context, int64) error
	OpenPullRequest(context.Context, openPullRequestParams) (string, error)
}

// github implements the githuber interface.
var _ githuber = (*github)(nil)

type github struct {
	client   *gogh.Client
	owner    string
	repoName string
}

// NewGithub returns a new Github client to interact with a given repository.
func NewGithub(ctx context.Context, repoURL string, pat stepconf.Secret) (*github, error) {
	// Initialize client for Github API.
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(pat)},
	)
	tokenClient := oauth2.NewClient(ctx, tokenSource)
	ghClient := gogh.NewClient(tokenClient)
	// Determine owner and repository name from url (all API requests need it).
	owner, repoName, err := githubOwnerRepo(repoURL)
	if err != nil {
		return nil, fmt.Errorf("owner and repo from url (%q): %w", repoURL, err)
	}
	return &github{
		client:   ghClient,
		owner:    owner,
		repoName: repoName,
	}, nil
}

func (gh github) AddKey(ctx context.Context, a []byte) (int64, error) {
	key, _, err := gh.client.Repositories.CreateKey(ctx, gh.owner, gh.repoName, &gogh.Key{
		Key:   gogh.String(string(a)),
		Title: gogh.String("Bitrise CI GitOps Integration"),
	})
	if err != nil {
		return 0, fmt.Errorf("create deploy key: %w", err)
	}
	return *key.ID, nil
}

func (gh github) DeleteKey(ctx context.Context, id int64) error {
	_, err := gh.client.Repositories.DeleteKey(ctx, gh.owner, gh.repoName, id)
	if err != nil {
		return fmt.Errorf("delete deploy key (%d): %w", id, err)
	}
	return nil
}

type openPullRequestParams struct {
	title string
	body  string
	head  string
	base  string
}

func (gh github) OpenPullRequest(ctx context.Context, p openPullRequestParams) (string, error) {
	// Title is required for PRs. Generate  one if it's omitted.
	if p.title == "" {
		p.title = "Merge " + p.head
	}
	req := &gogh.NewPullRequest{
		Title: gogh.String(p.title),
		Body:  gogh.String(p.body),
		Head:  gogh.String(p.head),
		Base:  gogh.String(p.base),
	}
	pr, _, err := gh.client.PullRequests.Create(ctx, gh.owner, gh.repoName, req)
	if err != nil {
		return "", fmt.Errorf("create: %w", err)
	}
	return *pr.HTMLURL, nil
}

func githubOwnerRepo(s string) (string, string, error) {
	// Trim prefix.
	prefix := "git@github.com:"
	if !strings.HasPrefix(s, prefix) {
		return "", "", fmt.Errorf("must start with %q", prefix)
	}
	s = strings.TrimPrefix(s, prefix)

	// Trim suffix.
	suffix := ".git"
	if !strings.HasSuffix(s, suffix) {
		return "", "", fmt.Errorf("must end with %q", suffix)
	}
	s = strings.TrimSuffix(s, suffix)

	// Split remaining URL for owner and repository name.
	a := strings.Split(s, "/")
	if len(a) != 2 {
		return "", "", fmt.Errorf("must separate owner from repo with one /")
	}
	return a[0], a[1], nil
}
