package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func (cfg *config) listExamples() ([]string, error) {
	groups, err := cfg.examplesBySource()
	if err != nil {
		return nil, err
	}
	var all []string
	for src, names := range groups {
		for _, name := range names {
			all = append(all, src+"/"+name)
		}
	}
	sort.Strings(all)
	return all, nil
}

func (cfg *config) examplesBySource() (map[string][]string, error) {
	return map[string][]string{
		"deckviz": collectExampleNames(cfg.deckviz.dir),
		"dubois":  collectExampleNames(cfg.dubois.dir),
	}, nil
}

func (cfg *config) runExamples(ctx context.Context, examples []string) (map[string]string, error) {
	results := make(map[string]string)
	for _, raw := range examples {
		source, name := parseExample(raw)
		dir, err := cfg.exampleDir(source, name)
		if err != nil {
			return nil, err
		}
		dshPath := filepath.Join(dir, name+".dsh")
		if _, err := os.Stat(dshPath); err != nil {
			fmt.Printf("Skipping %s: %v\n", normalizeExampleName(raw), err)
			continue
		}

		if err := cfg.runTool(ctx, dir, "dshlint", name+".dsh"); err != nil {
			return nil, err
		}

		xmlPath := filepath.Join(dir, name+".xml")
		if err := cfg.renderDeck(ctx, dir, name+".dsh", xmlPath); err != nil {
			return nil, err
		}
		results[normalizeExampleName(raw)] = xmlPath
	}

	if len(results) == 0 {
		return nil, errors.New("no examples rendered")
	}
	return results, nil
}

func (cfg *config) renderDeck(ctx context.Context, dir, script, output string) error {
	fmt.Printf("Rendering %s -> %s\n", script, output)
	deckshPath, err := cfg.resolveBinary("decksh")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()

	cmd := exec.CommandContext(ctx, deckshPath, script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DECKFONTS="+cfg.deckfontsEnv)
	cmd.Stdout = file
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (cfg *config) runTool(ctx context.Context, dir, tool, arg string) error {
	path, err := cfg.resolveBinary(tool)
	if err != nil {
		return err
	}
	fmt.Printf("Linting %s/%s\n", dir, arg)
	cmd := exec.CommandContext(ctx, path, arg)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DECKFONTS="+cfg.deckfontsEnv)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (cfg *config) resolveBinary(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	path := filepath.Join(cfg.goBinDir, name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("%s not found in PATH or %s", name, cfg.goBinDir)
}

func (cfg *config) exampleDir(source, name string) (string, error) {
	switch source {
	case "deckviz":
		return filepath.Join(cfg.deckviz.dir, name), nil
	case "dubois":
		return filepath.Join(cfg.dubois.dir, name), nil
	default:
		return "", fmt.Errorf("unknown example source %q", source)
	}
}

func (cfg *config) exampleCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	groups, err := cfg.examplesBySource()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	suggestions := make(map[string]struct{})

	if strings.Contains(toComplete, "/") {
		parts := strings.SplitN(toComplete, "/", 2)
		src := parts[0]
		partial := ""
		if len(parts) > 1 {
			partial = parts[1]
		}
		if names, ok := groups[src]; ok {
			for _, name := range names {
				if strings.HasPrefix(name, partial) {
					suggestions[src+"/"+name] = struct{}{}
				}
			}
		}
	} else {
		lower := strings.ToLower(toComplete)
		for src, names := range groups {
			prefix := src + "/"
			if strings.HasPrefix(strings.ToLower(prefix), lower) || toComplete == "" {
				suggestions[prefix] = struct{}{}
			}
			for _, name := range names {
				candidate := src + "/" + name
				if strings.HasPrefix(strings.ToLower(candidate), lower) || strings.HasPrefix(strings.ToLower(name), lower) {
					suggestions[candidate] = struct{}{}
					if src == "deckviz" {
						suggestions[name] = struct{}{}
					}
				}
			}
		}
	}

	var matches []string
	for suggestion := range suggestions {
		matches = append(matches, suggestion)
	}
	sort.Strings(matches)
	return matches, cobra.ShellCompDirectiveNoFileComp
}

func collectExampleNames(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		out = append(out, entry.Name())
	}
	sort.Strings(out)
	return out
}
