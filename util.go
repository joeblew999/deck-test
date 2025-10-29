package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Utility functions

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func resolveGoBin(goCmd string) (string, error) {
	if bin := strings.TrimSpace(runGoEnv(goCmd, "GOBIN")); bin != "" {
		return absPath(bin)
	}
	gopath := strings.TrimSpace(runGoEnv(goCmd, "GOPATH"))
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.New("unable to determine GOBIN; set GOBIN or GOPATH")
		}
		return filepath.Join(home, "go", "bin"), nil
	}
	first := strings.Split(gopath, string(os.PathListSeparator))[0]
	if first == "" {
		return "", errors.New("GOPATH is empty; set GOBIN explicitly")
	}
	return filepath.Join(first, "bin"), nil
}

func runGoEnv(goCmd, key string) string {
	cmd := exec.Command(goCmd, "env", key)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func absPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is empty")
	}
	return filepath.Abs(filepath.Clean(path))
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}
	return absPath(path)
}

func (cfg *config) parseExample(raw string) (source, name string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "deckviz", ""
	}

	// Handle filesystem paths like ".data/dubois-data-portraits/plate01"
	// Strip .data/ prefix if present
	if strings.HasPrefix(raw, dataDir+"/") {
		raw = strings.TrimPrefix(raw, dataDir+"/")
	}

	if strings.Contains(raw, "/") {
		parts := strings.SplitN(raw, "/", 2)
		src := parts[0]
		exampleName := strings.TrimSpace(parts[1])

		// Check if src is a directory name (e.g., "dubois-data-portraits")
		// and map it to logical repo name (e.g., "dubois")
		if repoName := cfg.getRepoNameByDir(src); repoName != "" {
			return repoName, exampleName
		}

		// Otherwise use as-is (already a logical name like "deckviz" or "dubois")
		if src == "" {
			src = "deckviz"
		}
		return src, exampleName
	}

	// No slash, default to deckviz
	return "deckviz", raw
}

func (cfg *config) normalizeExampleName(raw string) string {
	source, name := cfg.parseExample(raw)
	if source == "deckviz" {
		return "deckviz/" + name
	}
	return source + "/" + name
}
