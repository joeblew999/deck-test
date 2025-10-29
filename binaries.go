package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (cfg *config) resolveBinary(name string) (string, error) {
	// Check dist/ directory first (our downloaded binaries)
	distPath := cfg.getBinaryPath(name)
	if _, err := os.Stat(distPath); err == nil {
		return distPath, nil
	}

	// Fallback to PATH
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	// Finally check goBinDir
	path := cfg.getGoBinPath(name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("%s not found in %s, PATH, or %s", name, cfg.distDir, cfg.goBinDir)
}

func (cfg *config) ensureBins(ctx context.Context) error {
	// Download from GitHub releases only
	return cfg.downloadReleaseBinaries(ctx)
}

func (cfg *config) ensureGhCli(ctx context.Context) error {
	// Check if gh CLI is already installed
	if _, err := exec.LookPath("gh"); err == nil {
		return nil // Already installed
	}

	// Install gh CLI via go install
	fmt.Println("gh CLI not found, installing via go install...")
	installCmd := exec.CommandContext(ctx, cfg.goCmd, "install", "github.com/cli/cli/v2/cmd/gh@latest")
	installCmd.Env = append(os.Environ(), "GOBIN="+cfg.goBinDir)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install gh CLI: %w", err)
	}
	fmt.Println("✓ gh CLI installed successfully")

	// Update PATH to include GOBIN
	os.Setenv("PATH", cfg.goBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return nil
}

func (cfg *config) downloadReleaseBinaries(ctx context.Context) error {
	// Ensure gh CLI is installed
	if err := cfg.ensureGhCli(ctx); err != nil {
		return err
	}

	// Get latest release info
	fmt.Println("Checking for latest release...")
	listCmd := exec.CommandContext(ctx, "gh", "release", "list", "--limit", "1")
	output, err := listCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	// Parse release tag from output (format: "TITLE\tTYPE\tTAG\tDATE" - tab separated)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("no releases found")
	}
	// Split by tab to get fields
	fields := strings.Split(lines[0], "\t")
	if len(fields) < 3 {
		return fmt.Errorf("failed to parse release info")
	}
	releaseTag := strings.TrimSpace(fields[2]) // TAG is the 3rd field
	fmt.Printf("Latest release: %s\n", releaseTag)

	// Get release published/created time (use createdAt since publishedAt may be null for drafts)
	viewCmd := exec.CommandContext(ctx, "gh", "release", "view", releaseTag, "--json", "createdAt", "-q", ".createdAt")
	timeOutput, err := viewCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get release time: %w", err)
	}
	timeStr := strings.TrimSpace(string(timeOutput))
	if timeStr == "" || timeStr == "null" {
		// If no timestamp available, skip timestamp check and download everything
		fmt.Println("No release timestamp available, downloading all binaries...")
	}

	var releaseTime time.Time
	if timeStr != "" && timeStr != "null" {
		releaseTime, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return fmt.Errorf("failed to parse release time: %w", err)
		}
	}

	// Create dist directory if it doesn't exist
	if err := os.MkdirAll(cfg.distDir, 0755); err != nil {
		return fmt.Errorf("create dist dir: %w", err)
	}

	// Download native binaries for current platform
	downloaded := 0
	skipped := 0
	for _, spec := range cfg.toolchain {
		filename := cfg.buildFilename(spec.name, targetNative)
		destPath := filepath.Join(cfg.distDir, filename)

		// Check if local binary exists and compare timestamps (if available)
		fileInfo, err := os.Stat(destPath)
		if err == nil && !releaseTime.IsZero() {
			// File exists and we have a release time - check if local is newer
			localModTime := fileInfo.ModTime()
			if localModTime.After(releaseTime) {
				fmt.Printf("✓ %s is up to date (local is newer)\n", filename)
				skipped++
				continue
			}
			fmt.Printf("⟳ %s needs update (release is newer)\n", filename)
		} else if err == nil {
			// File exists but no release time - skip if file exists
			fmt.Printf("✓ %s already exists (no timestamp to compare)\n", filename)
			skipped++
			continue
		}

		fmt.Printf("Downloading %s...\n", filename)
		downloadCmd := exec.CommandContext(ctx, "gh", "release", "download", releaseTag, "-p", filename, "-D", cfg.distDir, "--clobber")
		downloadCmd.Stdout = os.Stdout
		downloadCmd.Stderr = os.Stderr
		if err := downloadCmd.Run(); err != nil {
			fmt.Printf("⚠ Failed to download %s: %v\n", filename, err)
			continue
		}

		// Make executable
		if err := os.Chmod(destPath, 0755); err != nil {
			fmt.Printf("⚠ Failed to chmod %s: %v\n", filename, err)
		}

		downloaded++
		fmt.Printf("✓ Downloaded %s\n", filename)
	}

	if downloaded == 0 && skipped > 0 {
		fmt.Println("All binaries are up to date")
	} else if downloaded > 0 {
		fmt.Printf("✓ Downloaded %d binaries (%d up to date)\n", downloaded, skipped)
	}

	return nil
}
