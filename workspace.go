package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Workspace management functions

func (cfg *config) ensureWorkspace(ctx context.Context) error {
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("create %s dir: %w", srcDir, err)
	}

	workFile := filepath.Join(srcDir, "go.work")

	// Build workspace content
	var dirs []string
	dirs = append(dirs, "..") // Parent directory (deck-test)

	for _, repo := range cfg.repos {
		if !repo.isData {
			repoName := filepath.Base(repo.dir)
			dirs = append(dirs, "./"+repoName)
		}
	}

	// Create go.work file
	content := "go 1.25\n\n"
	for _, dir := range dirs {
		content += fmt.Sprintf("use %s\n", dir)
	}

	if err := os.WriteFile(workFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write go.work: %w", err)
	}

	fmt.Printf("✓ Created %s/go.work workspace file\n", srcDir)
	return nil
}
