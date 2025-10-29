package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/spf13/cobra"
)

func newRootCommand(cfg *config) *cobra.Command {
	root := &cobra.Command{
		Use:           "decktool",
		Short:         "Helper CLI for deck examples",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return cfg.finalize()
		},
	}

	// Note: Repo-specific flags removed for simplicity
	// Use environment variables instead (DECKVIZ_DIR, DECKFONTS_DIR, etc.)

	root.AddCommand(newEnsureCommand(cfg))
	root.AddCommand(newExamplesCommand(cfg))
	root.AddCommand(newRunCommand(cfg))
	root.AddCommand(newViewCommand(cfg))
	root.AddCommand(newCompletionCommand(root))
	root.AddCommand(newSetupCommand(cfg))
	root.AddCommand(newDevBuildCommand(cfg))
	root.AddCommand(newDevReleaseCommand(cfg))
	root.AddCommand(newDevCleanCommand(cfg))

	return root
}

func newEnsureCommand(cfg *config) *cobra.Command {
	return &cobra.Command{
		Use:   "ensure",
		Short: "Install Go binaries and sync repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if err := cfg.ensureBins(ctx); err != nil {
				return err
			}
			if err := cfg.ensureRepos(ctx); err != nil {
				return err
			}
			fmt.Println("Tooling and repositories are up to date.")
			return nil
		},
	}
}

func newExamplesCommand(cfg *config) *cobra.Command {
	return &cobra.Command{
		Use:   "examples",
		Short: "List available examples",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cfg.ensureRepos(cmd.Context()); err != nil {
				return err
			}
			examples, err := cfg.listExamples()
			if err != nil {
				return err
			}
			for _, ex := range examples {
				fmt.Println(ex)
			}
			return nil
		},
		ValidArgsFunction: cfg.exampleCompletion,
	}
}

func newRunCommand(cfg *config) *cobra.Command {
	return &cobra.Command{
		Use:               "run [example]...",
		Short:             "Lint and render one or more examples",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: cfg.exampleCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cfg.ensureBins(cmd.Context()); err != nil {
				return err
			}
			if err := cfg.ensureRepos(cmd.Context()); err != nil {
				return err
			}
			results, err := cfg.runExamples(cmd.Context(), args)
			if err != nil {
				return err
			}
			var keys []string
			for k := range results {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, name := range keys {
				fmt.Printf("%s -> %s\n", name, results[name])
			}
			return nil
		},
	}
}

func newViewCommand(cfg *config) *cobra.Command {
	return &cobra.Command{
		Use:               "view [example]",
		Short:             "Render and open an example in ebdeck",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.exampleCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cfg.ensureBins(cmd.Context()); err != nil {
				return err
			}
			if err := cfg.ensureRepos(cmd.Context()); err != nil {
				return err
			}
			results, err := cfg.runExamples(cmd.Context(), args)
			if err != nil {
				return err
			}
			xmlPath, ok := results[cfg.normalizeExampleName(args[0])]
			if !ok {
				return fmt.Errorf("rendered XML not found for %q", args[0])
			}

			// Get example directory to run ebdeck from there (for relative paths in XML)
			source, name := cfg.parseExample(args[0])
			exampleDir, err := cfg.getExampleDir(source, name)
			if err != nil {
				return err
			}

			ebdeckPath, err := cfg.resolveBinary("ebdeck")
			if err != nil {
				return err
			}
			viewCmd := exec.CommandContext(cmd.Context(), ebdeckPath, xmlPath)
			viewCmd.Dir = exampleDir
			viewCmd.Env = append(os.Environ(), "DECKFONTS="+cfg.fontsDir)
			viewCmd.Stdout = os.Stdout
			viewCmd.Stderr = os.Stderr
			return viewCmd.Run()
		},
	}
}
