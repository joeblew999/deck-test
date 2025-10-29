package main

import (
	"fmt"
	"path/filepath"
)

// Path helper functions

func (cfg *config) getDistGlob() string {
	return filepath.Join(filepath.Base(cfg.distDir), "*")
}

func (cfg *config) getGoBinPath(name string) string {
	return filepath.Join(cfg.goBinDir, name)
}

func (cfg *config) getExampleDir(source, name string) (string, error) {
	switch source {
	case "deckviz":
		return filepath.Join(cfg.repos["deckviz"].dir, name), nil
	case "dubois":
		return filepath.Join(cfg.repos["dubois"].dir, name), nil
	default:
		return "", fmt.Errorf("unknown example source %q", source)
	}
}

func (cfg *config) getExampleDshPath(dir, name string) string {
	return filepath.Join(dir, name+".dsh")
}

func (cfg *config) getExampleXmlPath(dir, name string) string {
	return filepath.Join(dir, name+".xml")
}

func getShellCompletionPath(home, shell string) (string, error) {
	switch shell {
	case "zsh":
		return filepath.Join(home, ".decktool", "completions", "_decktool"), nil
	case "bash":
		return filepath.Join(home, ".decktool", "completions", "decktool.bash"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "completions", "decktool.fish"), nil
	case "powershell":
		return filepath.Join(home, "Documents", "PowerShell", "decktool.ps1"), nil
	default:
		return "", nil
	}
}

func getShellRCPath(home, shell string) string {
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "bash":
		return filepath.Join(home, ".bashrc")
	case "powershell":
		return filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	default:
		return ""
	}
}

func (cfg *config) getRepoNameByDir(dirName string) string {
	for name, repo := range cfg.repos {
		if filepath.Base(repo.dir) == dirName {
			return name
		}
	}
	return ""
}
