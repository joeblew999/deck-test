package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Release operations

// Release operations

func (cfg *config) createGithubRelease(ctx context.Context, version string, prerelease bool) error {
	// Ensure gh CLI is installed
	if err := cfg.ensureGhCli(ctx); err != nil {
		return err
	}

	// Check authentication
	authCmd := exec.CommandContext(ctx, "gh", "auth", "status")
	if err := authCmd.Run(); err != nil {
		fmt.Println("Please authenticate with GitHub:")
		loginCmd := exec.CommandContext(ctx, "gh", "auth", "login")
		loginCmd.Stdin = os.Stdin
		loginCmd.Stdout = os.Stdout
		loginCmd.Stderr = os.Stderr
		if err := loginCmd.Run(); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Find all binaries in dist directory
	binaries, err := filepath.Glob(cfg.getDistGlob())
	if err != nil {
		return fmt.Errorf("failed to glob binaries: %w", err)
	}
	if len(binaries) == 0 {
		return fmt.Errorf("no binaries found in %s (run dev-build first)", cfg.distDir)
	}

	// Create release
	fmt.Printf("Creating release %s...\n", version)
	releaseArgs := []string{"release", "create", version}
	if prerelease {
		releaseArgs = append(releaseArgs, "--prerelease")
	}
	releaseArgs = append(releaseArgs, "--title", version)
	releaseArgs = append(releaseArgs, "--notes", fmt.Sprintf("Release %s\n\nBuilt with decktool", version))
	releaseArgs = append(releaseArgs, binaries...)

	releaseCmd := exec.CommandContext(ctx, "gh", releaseArgs...)
	releaseCmd.Stdout = os.Stdout
	releaseCmd.Stderr = os.Stderr
	if err := releaseCmd.Run(); err != nil {
		return fmt.Errorf("release creation failed: %w", err)
	}

	fmt.Printf("✓ Release %s created with %d binaries\n", version, len(binaries))
	return nil
}

func (cfg *config) generateReleaseVersion() string {
	return fmt.Sprintf("dev-%s", time.Now().Format("20060102-150405"))
}
