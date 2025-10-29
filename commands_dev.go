package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Dev commands

// Dev commands

func newDevBuildCommand(cfg *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev-build",
		Short: "Build all deck binaries for native, WASM, and WASI targets",
		Long: `Build all deck binaries for all targets (native, wasm, wasi).

Examples:
  decktool dev-build`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Ensure repos are synced
			fmt.Println("Syncing build repositories...")
			if err := cfg.ensureBuildRepos(ctx); err != nil {
				return fmt.Errorf("sync build repos: %w", err)
			}
			fmt.Println("Creating go.work workspace...")
			if err := cfg.ensureWorkspace(ctx); err != nil {
				return fmt.Errorf("create workspace: %w", err)
			}

			// Build all targets to dist directory
			buildTargets := []buildTarget{targetNative, targetWASM, targetWASI}

			fmt.Printf("Building %d binaries for targets: %v\n", len(cfg.toolchain), buildTargets)
			results, err := cfg.buildAll(ctx, buildTargets, cfg.distDir)
			if err != nil {
				return err
			}

			// Report results
			fmt.Println("\n=== Build Results ===")
			successes := 0
			failures := 0
			skipped := 0
			for _, result := range results {
				if result.err != nil {
					if strings.Contains(result.err.Error(), "not supported") {
						fmt.Printf("⊘ %s: %v\n", result.binary, result.err)
						skipped++
					} else {
						fmt.Printf("✗ %s: %v\n", result.binary, result.err)
						failures++
					}
				} else {
					fmt.Printf("✓ %s\n", result.path)
					successes++
				}
			}
			fmt.Printf("\nTotal: %d succeeded, %d failed, %d skipped\n", successes, failures, skipped)

			if failures > 0 {
				return fmt.Errorf("some builds failed")
			}

			return nil
		},
	}
	return cmd
}

func newDevReleaseCommand(cfg *config) *cobra.Command {
	var skipBuild bool
	var prerelease bool
	var version string

	cmd := &cobra.Command{
		Use:   "dev-release",
		Short: "Create a GitHub release with built binaries",
		Long: `Create a GitHub release and upload all binaries from dist/ directory.

By default, creates a timestamped prerelease (e.g., dev-20251029-143052).
Use --version to specify a custom version tag.

Examples:
  decktool dev-release                           # Auto-timestamped prerelease
  decktool dev-release --version=v0.1.0          # Official release
  decktool dev-release --version=v0.1.0-beta     # Beta prerelease
  decktool dev-release --skip-build              # Use existing dist/ binaries`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Build all binaries unless skipped
			if !skipBuild {
				fmt.Println("Building all binaries...")
				buildTargets := []buildTarget{targetNative, targetWASM, targetWASI}
				results, err := cfg.buildAll(ctx, buildTargets, cfg.distDir)
				if err != nil {
					return fmt.Errorf("build failed: %w", err)
				}

				// Check for build failures
				failCount := 0
				for _, result := range results {
					if result.err != nil && !strings.Contains(result.err.Error(), "not supported") {
						failCount++
					}
				}
				if failCount > 0 {
					return fmt.Errorf("some builds failed, cannot create release")
				}
				fmt.Println("✓ Build completed")
			}

			// Generate version if not specified
			if version == "" {
				version = cfg.generateReleaseVersion()
				prerelease = true
			}

			// Create GitHub release
			return cfg.createGithubRelease(ctx, version, prerelease)
		},
	}
	cmd.Flags().BoolVar(&skipBuild, "skip-build", false, "skip building binaries, use existing dist/ files")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "mark as prerelease (default for auto-versioned releases)")
	cmd.Flags().StringVar(&version, "version", "", "version tag (default: auto-generated timestamp)")
	return cmd
}

func newDevCleanCommand(cfg *config) *cobra.Command {
	return &cobra.Command{
		Use:   "dev-clean",
		Short: "Remove all dot folders (.data, .src, .dist, .fonts) for fresh start",
		Long: `Remove all cached data folders including repositories, source code, built binaries, and fonts.

This is useful for starting fresh or troubleshooting issues.

Examples:
  decktool dev-clean`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Remove dot folders defined in config
			folders := []string{cfg.distDir, cfg.fontsDir}
			for _, repo := range cfg.repos {
				folders = append(folders, repo.dir)
			}

			for _, folder := range folders {
				if _, err := os.Stat(folder); err == nil {
					fmt.Printf("Removing %s...\n", folder)
					if err := os.RemoveAll(folder); err != nil {
						return fmt.Errorf("failed to remove %s: %w", folder, err)
					}
				}
			}

			fmt.Println("✓ All dot folders removed")
			return nil
		},
	}
}
