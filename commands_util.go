package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Utility commands

func newCompletionCommand(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(os.Stdout)
			case "zsh":
				return root.GenZshCompletion(os.Stdout)
			case "fish":
				return root.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
}

func newSetupCommand(cfg *config) *cobra.Command {
	defaultShell := detectShell()

	install := true
	compShell := defaultShell
	compOutput := ""
	sync := false
	local := ""

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install decktool binary and optionally emit shell completions",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if local != "" && !cmd.Flags().Changed("install") {
				install = false
			}

			if sync {
				if err := cfg.ensureBins(ctx); err != nil {
					return err
				}
				if err := cfg.ensureRepos(ctx); err != nil {
					return err
				}
			}

			binaryPath, err := cfg.buildSelf(ctx, local)
			if err != nil {
				return err
			}
			if install {
				if err := cfg.installSelf(ctx, binaryPath, local); err != nil {
					return err
				}
			}

			if compShell != "" {
				if err := cfg.writeCompletion(cmd, compShell, compOutput); err != nil {
					return err
				}
			} else if !install && binaryPath == "" {
				return errors.New("nothing to do: specify --install or --completions")
			}

			return nil
		},
	}
	cmd.Flags().BoolVar(&install, "install", true, "run `go install` for decktool")
	cmd.Flags().StringVar(&compShell, "completions", compShell, "generate completions for shell (bash|zsh|fish|powershell)")
	cmd.Flags().StringVar(&compOutput, "output", "", "write completions to file (default auto path)")
	cmd.Flags().BoolVar(&sync, "sync", false, "sync repositories and tooling before installing")
	cmd.Flags().StringVar(&local, "local", "", "e.g. --local=bin/decktool to place binary in repo")
	return cmd
}
