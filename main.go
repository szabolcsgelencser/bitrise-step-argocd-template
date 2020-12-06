package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-steplib/steps-activate-ssh-key/activatesshkey"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	valuesYAMLPath := os.Getenv("deployments_folder_path") + "/values.yaml"
	vars := parseVars(os.Getenv("vars"))
	deployKey := os.Getenv("deploy_ssh_key")
	deployKeyPath := os.Getenv("ssh_key_save_path")
	repoURL := os.Getenv("deploy_repository")
	pathInRepo := os.Getenv("deploy_path")

	if err := gitAddKey(deployKeyPath, stepconf.Secret(deployKey)); err != nil {
		return fmt.Errorf("git add key: %w", err)
	}

	dr, err := newDeployRepository(repoURL, pathInRepo)
	if err != nil {
		return fmt.Errorf("new deploy repository: %w", err)
	}
	defer dr.close()

	if err := dr.gitClone(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}
	if err := dr.renderValuesYAML(valuesYAMLPath, vars); err != nil {
		return fmt.Errorf("render values.yaml file: %w", err)
	}
	commitMessage := fmt.Sprintf("test run at %s", time.Now())
	if err := dr.pushChanges(commitMessage); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	return nil
}

func parseVars(s string) map[string]string {
	m := map[string]string{}
	s = strings.TrimLeft(s, "map[")
	s = strings.TrimRight(s, "]")
	for _, pair := range strings.Split(s, " ") {
		a := strings.Split(pair, ":")
		m[a[0]] = a[1]
	}
	return m
}

func gitAddKey(path string, key stepconf.Secret) error {
	if err := activatesshkey.Execute(activatesshkey.Config{
		SSHRsaPrivateKey:        key,
		SSHKeySavePath:          path,
		IsRemoveOtherIdentities: true,
	}); err != nil {
		return fmt.Errorf("activate ssh key: %w", err)
	}
	return nil
}

type deployRepository struct {
	repoURL    string
	localDir   string
	pathInRepo string
}

func newDeployRepository(repoURL, pathInRepo string) (*deployRepository, error) {
	localDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, fmt.Errorf("create temporary directory: %w", err)
	}
	return &deployRepository{
		repoURL:    repoURL,
		localDir:   localDir,
		pathInRepo: pathInRepo,
	}, nil
}

func (dr deployRepository) close() {
	os.Remove(dr.localDir)
}

func (dr deployRepository) renderValuesYAML(tplPath string, vars interface{}) error {
	t, err := template.ParseFiles(tplPath)
	if err != nil {
		return fmt.Errorf("parse template %q: %w", tplPath, err)
	}
	renderPath := dr.localDir + "/" + dr.pathInRepo + "/values.yaml"
	f, err := os.Create(renderPath)
	if err != nil {
		return fmt.Errorf("create file to render: %w", err)
	}
	if err := t.Execute(f, vars); err != nil {
		return fmt.Errorf("execute values.yaml template: %w", err)
	}
	return nil
}

func (dr deployRepository) gitClone() error {
	_, err := runCommand("git", "clone", dr.repoURL, dr.localDir)
	return err
}

func (dr deployRepository) pushChanges(message string) error {
	startingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current dir: %w", err)
	}
	if err := os.Chdir(dr.localDir); err != nil {
		return fmt.Errorf("change dir ot %q: %w", dr.localDir, err)
	}
	defer os.Chdir(startingDir)

	status, err := runCommand("git", "status")
	if err != nil {
		return err
	}
	if strings.Contains(status, "nothing to commit, working tree clean") {
		fmt.Println("Deployment configuration didn't change, nothing to push.")
		return nil
	}

	gitArgs := [][]string{
		{"add", "--all"},
		{"commit", "-m", message},
		{"push"},
	}
	for _, a := range gitArgs {
		if _, err := runCommand("git", a...); err != nil {
			return err
		}
	}
	return nil
}

func runCommand(cmd string, args ...string) (string, error) {
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("run command %v: %w (output: %s)", args, err, out)
	}
	return string(out), nil
}
