package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Build-related functions

func (cfg *config) buildBinary(ctx context.Context, spec binSpec, target buildTarget, outputDir string) buildResult {
	result := buildResult{
		binary: spec.name,
		target: target,
	}

	// Check target support
	if target == targetWASM && !spec.wasmSupport {
		result.err = fmt.Errorf("WASM not supported (requires %v)", getRequirement(spec))
		return result
	}
	if target == targetWASI && !spec.wasiSupport {
		result.err = fmt.Errorf("WASI not supported (requires %v)", getRequirement(spec))
		return result
	}

	// Build filename with descriptive suffix (flat structure for GitHub releases)
	filename := cfg.buildFilename(spec.name, target)
	outPath := filepath.Join(outputDir, filename)
	result.path = outPath

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.err = fmt.Errorf("mkdir: %w", err)
		return result
	}

	// Make output path absolute (needed because we run from srcDir)
	absOutPath, err := filepath.Abs(outPath)
	if err != nil {
		result.err = fmt.Errorf("abs path: %w", err)
		return result
	}

	// Build from srcDir using go.work
	fmt.Printf("Building %s for %s...\n", spec.name, target)

	cmd := exec.CommandContext(ctx, cfg.goCmd, "build", "-o", absOutPath, spec.pkg)
	cmd.Dir = srcDir // Run from workspace directory

	// Set cross-compilation environment
	goos, goarch := target.buildEnv()
	env := os.Environ()
	if goos != "" {
		env = append(env, "GOOS="+goos)
	}
	if goarch != "" {
		env = append(env, "GOARCH="+goarch)
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		result.err = fmt.Errorf("build failed: %w", err)
		return result
	}

	fmt.Printf("✓ Built %s\n", filename)
	return result
}

func (cfg *config) buildFilename(name string, target buildTarget) string {
	switch target {
	case targetWASM:
		return fmt.Sprintf("%s-wasm.wasm", name)
	case targetWASI:
		return fmt.Sprintf("%s-wasi.wasm", name)
	default: // native
		goos := runtime.GOOS
		goarch := runtime.GOARCH
		ext := ""
		if goos == "windows" {
			ext = ".exe"
		}
		return fmt.Sprintf("%s-%s-%s%s", name, goos, goarch, ext)
	}
}

func (cfg *config) buildAll(ctx context.Context, targets []buildTarget, outputDir string) ([]buildResult, error) {
	var results []buildResult
	for _, spec := range cfg.toolchain {
		for _, target := range targets {
			result := cfg.buildBinary(ctx, spec, target, outputDir)
			results = append(results, result)
		}
	}
	return results, nil
}

func getRequirement(spec binSpec) string {
	if spec.requiresUI {
		return "UI/graphics"
	}
	return "platform support"
}

func (cfg *config) getBinaryPath(name string) string {
	return filepath.Join(cfg.distDir, name+"-"+runtime.GOOS+"-"+runtime.GOARCH)
}

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

func (cfg *config) downloadReleaseBinaries(ctx context.Context) error {
	// Check if gh CLI is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found. Install it with: brew install gh")
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
