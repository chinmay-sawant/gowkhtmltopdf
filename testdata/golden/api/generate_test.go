package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTemplatePathsFromAPIDirectory(t *testing.T) {
	t.Parallel()

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	input, output, err := resolveTemplatePaths()
	if err != nil {
		t.Fatal(err)
	}

	wantInput := filepath.Join(workingDir, inputName)
	wantOutput := filepath.Join(workingDir, outputName)

	if input != wantInput || output != wantOutput {
		t.Fatalf("resolved paths = %q, %q; want %q, %q", input, output, wantInput, wantOutput)
	}
}
