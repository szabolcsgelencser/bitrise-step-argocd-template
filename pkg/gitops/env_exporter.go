package gitops

import (
	"fmt"
	"os/exec"
)

type envExporter func(name, value string) error

// Ensure EnvmanExport is an envExporter.
var _ envExporter = EnvmanExport

// EnvmanExport exports an environment variable using envman.
func EnvmanExport(name, value string) error {
	c := exec.Command("envman", "add", "--key", name, "--value", value)
	if err := c.Run(); err != nil {
		return fmt.Errorf("export %s with envman: %w", name, err)
	}
	return nil
}
