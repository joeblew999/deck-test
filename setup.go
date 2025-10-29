package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func (cfg *config) buildSelf(ctx context.Context, local string) (string, error) {
	local = strings.TrimSpace(local)
	if local == "" {
		return "", nil
	}
	abs, err := expandPath(local)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	fmt.Printf("Building decktool to %s\n", abs)
	cmd := exec.CommandContext(ctx, cfg.goCmd, "build", "-o", abs, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return abs, nil
}

func (cfg *config) installSelf(ctx context.Context, builtPath, local string) error {
	if strings.TrimSpace(local) != "" {
		fmt.Printf("Local decktool binary located at %s\n", builtPath)
		return nil
	}
	fmt.Println("Installing decktool into GOBIN")
	cmd := exec.CommandContext(ctx, cfg.goCmd, "install", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (cfg *config) writeCompletion(cmd *cobra.Command, shell, output string) error {
	root := cmd.Root()
	var buf bytes.Buffer
	var err error
	switch shell {
	case "bash":
		err = root.GenBashCompletion(&buf)
	case "zsh":
		err = root.GenZshCompletion(&buf)
	case "fish":
		err = root.GenFishCompletion(&buf, true)
	case "powershell":
		err = root.GenPowerShellCompletionWithDesc(&buf)
	default:
		return fmt.Errorf("unsupported shell %q", shell)
	}
	if err != nil {
		return err
	}

	output = strings.TrimSpace(output)
	if output == "" || output == "-" {
		if def, derr := defaultCompletionPath(shell); derr == nil && def != "" {
			output = def
		}
	}

	if output == "" || output == "-" {
		_, err = os.Stdout.Write(buf.Bytes())
		return err
	}

	abs, err := expandPath(output)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(abs, buf.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Printf("Wrote %s completions to %s\n", shell, abs)

	if rc, snippet := defaultRCConfig(shell, abs); rc != "" && snippet != "" {
		if err := ensureShellSnippet(rc, snippet); err != nil {
			fmt.Fprintf(os.Stderr, "warning: unable to update %s automatically (%v)\n", rc, err)
		} else {
			fmt.Printf("Updated %s to source completion script.\n", rc)
		}
	}

	return nil
}

func detectShell() string {
	env := strings.TrimSpace(os.Getenv("SHELL"))
	if env == "" {
		if out := strings.TrimSpace(runGoEnv("go", "SHELL")); out != "" {
			env = out
		}
	}
	if env == "" {
		return ""
	}
	return filepath.Base(env)
}

func defaultCompletionPath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return getShellCompletionPath(home, shell)
}

func defaultRCConfig(shell, completionPath string) (rcPath, snippet string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", ""
	}
	rc := getShellRCPath(home, shell)
	if rc == "" {
		return "", ""
	}

	switch shell {
	case "zsh":
		snippet = fmt.Sprintf("\n# decktool completions\nif [ -f %q ]; then\n  source %q\nfi\n", completionPath, completionPath)
	case "bash":
		snippet = fmt.Sprintf("\n# decktool completions\nif [ -f %q ]; then\n  . %q\nfi\n", completionPath, completionPath)
	case "powershell":
		snippet = fmt.Sprintf("\n# decktool completions\nif (Test-Path %q) { . %q }\n", completionPath, completionPath)
	}
	return rc, snippet
}

func ensureShellSnippet(rcPath, snippet string) error {
	abs, err := expandPath(rcPath)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return err
			}
			return os.WriteFile(abs, []byte(snippet), 0o644)
		}
		return err
	}
	if strings.Contains(string(data), snippet) {
		return nil
	}
	f, err := os.OpenFile(abs, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(snippet)
	return err
}
