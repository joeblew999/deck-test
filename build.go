package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

