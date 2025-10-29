package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
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
	result := make(map[string][]string)
	for name, repo := range cfg.repos {
		// Only include data repos that contain examples
		if !repo.isData {
			continue
		}
		result[name] = collectExampleNames(repo.dir)
	}
	return result, nil
}

func (cfg *config) runExamples(ctx context.Context, examples []string) (map[string]string, error) {
	// Set DECKFONTS for all child processes
	// NOTE: Due to a quirk with how Go's os.Setenv() interacts with some binaries,
	// DECKFONTS may need to be exported in the shell before running decktool for
	// the view/run commands to work properly. The ensure command prints the export.
	oldDeckfonts := os.Getenv("DECKFONTS")
	os.Setenv("DECKFONTS", cfg.fontsDir)
	defer func() {
		if oldDeckfonts != "" {
			os.Setenv("DECKFONTS", oldDeckfonts)
		} else {
			os.Unsetenv("DECKFONTS")
		}
	}()

	results := make(map[string]string)
	for _, raw := range examples {
		source, name := cfg.parseExample(raw)
		dir, err := cfg.getExampleDir(source, name)
		if err != nil {
			return nil, err
		}
		dshPath := cfg.getExampleDshPath(dir, name)
		if _, err := os.Stat(dshPath); err != nil {
			fmt.Printf("Skipping %s: %v\n", cfg.normalizeExampleName(raw), err)
			continue
		}

		if err := cfg.runTool(ctx, dir, "dshlint", name+".dsh"); err != nil {
			return nil, err
		}

		xmlPath := cfg.getExampleXmlPath(dir, name)
		if err := cfg.renderDeck(ctx, dir, name+".dsh", xmlPath); err != nil {
			return nil, err
		}
		results[cfg.normalizeExampleName(raw)] = xmlPath
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
