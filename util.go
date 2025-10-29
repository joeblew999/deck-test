package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

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
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	return absPath(path)
}

func parseExample(raw string) (source, name string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "deckviz", ""
	}
	if strings.Contains(raw, "/") {
		parts := strings.SplitN(raw, "/", 2)
		src := parts[0]
		if src == "" {
			src = "deckviz"
		}
		return src, strings.TrimSpace(parts[1])
	}
	return "deckviz", raw
}

func normalizeExampleName(raw string) string {
	source, name := parseExample(raw)
	if source == "deckviz" {
		return "deckviz/" + name
	}
	return source + "/" + name
}
