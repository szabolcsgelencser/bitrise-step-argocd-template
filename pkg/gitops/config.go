package gitops

import (
	"fmt"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
)

type config struct {
	// DeployRepositoryURL is the URL of the deployment (GitOps) repository.
	DeployRepositoryURL string `env:"deploy_repository_url,required"`
	// DeployFolder is the folder to render templates to in the deploy repository.
	DeployFolder string `env:"deploy_path,required"`
	// DeployBranch is the branch to render templates to in the deploy repository.
	DeployBranch string `env:"deploy_branch,required"`
	// PullRequest won't push to the branch. It will open a PR only instead.
	PullRequest bool `env:"pull_request"`
	// PullRequestTitle is the title of the opened pull request.
	PullRequestTitle string `env:"pull_request_title"`
	// PullRequestBody is the body of the opened pull request.
	PullRequestBody string `env:"pull_request_body"`
	// RawVars are unparsed version of `Vars` field (to-be-parsed manually).
	RawVars string `env:"vars"`
	// Vars are variables applied to the template files.
	Vars map[string]string
	// TemplatesFolder is the path to the deployment templates folder.
	TemplatesFolder string `env:"templates_folder_path,dir"`
	// DeployPAT is the Personal Access Token to interact with Github API.
	DeployPAT stepconf.Secret `env:"deploy_pat,required"`
	// CommitMessage is the created commit's message.
	CommitMessage string `env:"commit_message,required"`
}

// NewConfig returns a new configuration initialized from environment variables.
func NewConfig() (config, error) {
	var cfg config
	if err := stepconf.Parse(&cfg); err != nil {
		return config{}, fmt.Errorf("parse step config: %w", err)
	}
	cfg.Vars = parseMap(cfg.RawVars)
	return cfg, nil
}

// parseMap returns a deserialized map[string]string from a given string.
// Assumption: keys don't contain spaces, values can.
// (it cannot be confidently deserialized if we allow both)
func parseMap(s string) map[string]string {
	s = strings.TrimPrefix(s, "map[")
	s = strings.TrimSuffix(s, "]")

	m := map[string]string{}
	var key, value string
	var b strings.Builder
	for _, r := range s {
		switch r {
		case ':':
			if key != "" {
				m[key] = value
			}
			key = b.String()
			b.Reset()
			value = ""
		case ' ':
			value = appendWord(value, b.String())
			b.Reset()
		default:
			b.WriteRune(r)
		}
	}
	m[key] = appendWord(value, b.String())
	return m
}

func appendWord(sentence, word string) string {
	if sentence == "" {
		return word
	}
	return sentence + " " + word
}
