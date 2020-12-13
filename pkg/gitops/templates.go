package gitops

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
)

//go:generate moq -out templates_moq_test.go . renderAllFileser
type renderAllFileser interface {
	renderAllFiles() error
}

// templatesRenderer implements the renderAllFileser interface.
var _ sshKeyer = (*sshKey)(nil)

// TemplatesRenderer renders a folder of templates to a local repository.
type TemplatesRenderer struct {
	// Source folder of templates.
	SourceFolder string
	// Variables to substitute into the templates.
	Vars map[string]string
	// Destination repository for rendered files.
	DestinationRepo repositorier
	// Destination folder inside the repository for rendered files.
	DestinationFolder string
}

func (tr TemplatesRenderer) renderAllFiles() error {
	// Get all template file names from the source folder.
	files, err := ioutil.ReadDir(tr.SourceFolder)
	if err != nil {
		return fmt.Errorf("read files in %q: %w", tr.SourceFolder, err)
	}

	// Render templates one-by-one to the destinaton folder
	// (substituting variables given).
	for _, file := range files {
		if err := tr.renderFile(file.Name()); err != nil {
			return fmt.Errorf("render file %q: %w", file.Name(), err)
		}
	}
	return nil
}

func (tr TemplatesRenderer) renderFile(fileName string) error {
	// Parse template.
	sourceFilePath := filepath.Join(tr.SourceFolder, fileName)
	t, err := template.ParseFiles(sourceFilePath)
	if err != nil {
		return fmt.Errorf("parse template %q: %w", sourceFilePath, err)
	}

	// Create a file for the rendered template.
	destinationFilePath := filepath.Join(
		tr.DestinationRepo.localPath(), tr.DestinationFolder, fileName)
	f, err := os.Create(destinationFilePath)
	if err != nil {
		return fmt.Errorf("create destionation file: %w", err)
	}

	// Render the template to the previously created file.
	if err := t.Option("missingkey=error").Execute(f, tr.Vars); err != nil {
		return fmt.Errorf("execute template %q: %w", sourceFilePath, err)
	}
	return nil
}
