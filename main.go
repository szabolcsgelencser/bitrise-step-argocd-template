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

	localDir, err := ioutil.TempDir("", "")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	// defer os.Remove(localDir)
	fmt.Printf("created tmp directory at: %s\n", localDir)

	if err := gitAddKey(deployKeyPath, stepconf.Secret(deployKey)); err != nil {
		return fmt.Errorf("git add key: %w", err)
	}
	if err := gitClone(repoURL, localDir); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}
	renderPath := localDir + "/" + pathInRepo + "/values.yaml"
	if err := renderValuesYAML(renderPath, valuesYAMLPath, vars); err != nil {
		return fmt.Errorf("render values.yaml file: %w", err)
	}
	commitMessage := fmt.Sprintf("test run at %s", time.Now())
	if err := gitPush(localDir, commitMessage); err != nil {
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

func renderValuesYAML(renderPath, tplPath string, vars interface{}) error {
	t, err := template.ParseFiles(tplPath)
	if err != nil {
		return fmt.Errorf("parse template %q: %w", tplPath, err)
	}
	f, err := os.Create(renderPath)
	if err != nil {
		return fmt.Errorf("create file to render: %w", err)
	}
	if err := t.Execute(f, vars); err != nil {
		return fmt.Errorf("execute values.yaml template: %w", err)
	}
	return nil
}

func gitAddKey(path string, key stepconf.Secret) error {
	if err := activatesshkey.Execute(activatesshkey.Config{
		SSHRsaPrivateKey:        key,
		SSHKeySavePath:          path,
		IsRemoveOtherIdentities: false,
	}); err != nil {
		return fmt.Errorf("activate ssh key: %w", err)
	}
	return nil
}

func gitClone(repoURL, localDir string) error {
	return runCommand("git", "clone", repoURL, localDir)
}

func gitPush(localDir, message string) error {
	startingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get current dir: %w", err)
	}
	if err := os.Chdir(localDir); err != nil {
		return fmt.Errorf("change dir ot %q: %w", localDir, err)
	}
	defer os.Chdir(startingDir)

	gitArgs := [][]string{
		{"add", "--all"},
		{"commit", "-m", message},
		{"push"},
	}
	for _, a := range gitArgs {
		if err := runCommand("git", a...); err != nil {
			return err
		}
	}
	return nil
}

func runCommand(cmd string, args ...string) error {
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("run command %v: %w (output: %s)", args, err, out)
	}
	return nil
}
