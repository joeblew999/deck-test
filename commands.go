package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

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

func newDevBuildCommand(cfg *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev-build",
		Short: "Build all deck binaries for native, WASM, and WASI targets",
		Long: `Build all deck binaries for all targets (native, wasm, wasi).

Examples:
  decktool dev-build`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Ensure repos are synced
			fmt.Println("Syncing build repositories...")
			if err := cfg.ensureBuildRepos(ctx); err != nil {
				return fmt.Errorf("sync build repos: %w", err)
			}
			fmt.Println("Creating go.work workspace...")
			if err := cfg.ensureWorkspace(ctx); err != nil {
				return fmt.Errorf("create workspace: %w", err)
			}

			// Build all targets to dist directory
			buildTargets := []buildTarget{targetNative, targetWASM, targetWASI}

			fmt.Printf("Building %d binaries for targets: %v\n", len(cfg.toolchain), buildTargets)
			results, err := cfg.buildAll(ctx, buildTargets, cfg.distDir)
			if err != nil {
				return err
			}

			// Report results
			fmt.Println("\n=== Build Results ===")
			successCount := 0
			failCount := 0
			skipCount := 0

			for _, result := range results {
				status := "✓"
				msg := result.path
				if result.err != nil {
					if strings.Contains(result.err.Error(), "not supported") {
						status = "⊘"
						skipCount++
						msg = result.err.Error()
					} else {
						status = "✗"
						failCount++
						msg = result.err.Error()
					}
				} else {
					successCount++
				}
				fmt.Printf("%s %s (%s): %s\n", status, result.binary, result.target, msg)
			}

			fmt.Printf("\nSummary: %d succeeded, %d failed, %d skipped\n", successCount, failCount, skipCount)

			if failCount > 0 {
				return fmt.Errorf("some builds failed")
			}

			return nil
		},
	}

	return cmd
}

func newDevReleaseCommand(cfg *config) *cobra.Command {
	var skipBuild bool
	var prerelease bool
	var version string

	cmd := &cobra.Command{
		Use:   "dev-release",
		Short: "Create a GitHub release with built binaries",
		Long: `Create a GitHub release and upload all binaries from dist/ directory.

By default, creates a timestamped prerelease (e.g., dev-20251029-143052).
Use --version to specify a custom version tag.

Examples:
  decktool dev-release                           # Auto-timestamped prerelease
  decktool dev-release --version=v0.1.0          # Official release
  decktool dev-release --version=v0.1.0-beta     # Beta prerelease
  decktool dev-release --skip-build              # Use existing dist/ binaries`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Build all binaries unless skipped
			if !skipBuild {
				fmt.Println("Building all binaries...")
				buildTargets := []buildTarget{targetNative, targetWASM, targetWASI}
				results, err := cfg.buildAll(ctx, buildTargets, cfg.distDir)
				if err != nil {
					return fmt.Errorf("build failed: %w", err)
				}

				// Check for build failures
				failCount := 0
				for _, result := range results {
					if result.err != nil && !strings.Contains(result.err.Error(), "not supported") {
						failCount++
					}
				}
				if failCount > 0 {
					return fmt.Errorf("some builds failed, cannot create release")
				}
				fmt.Println("✓ Build completed")
			}

			// Generate version if not specified
			if version == "" {
				version = fmt.Sprintf("dev-%s", time.Now().Format("20060102-150405"))
				prerelease = true
			}

			// Ensure gh CLI is installed
			if err := cfg.ensureGhCli(ctx); err != nil {
				return err
			}

			// Check authentication
			authCmd := exec.CommandContext(ctx, "gh", "auth", "status")
			if err := authCmd.Run(); err != nil {
				fmt.Println("Please authenticate with GitHub:")
				loginCmd := exec.CommandContext(ctx, "gh", "auth", "login")
				loginCmd.Stdin = os.Stdin
				loginCmd.Stdout = os.Stdout
				loginCmd.Stderr = os.Stderr
				if err := loginCmd.Run(); err != nil {
					return fmt.Errorf("authentication failed: %w", err)
				}
			}

			// Get repo name for release notes
			repoCmd := exec.CommandContext(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
			repoOutput, err := repoCmd.Output()
			if err != nil {
				return fmt.Errorf("failed to get repo info: %w", err)
			}
			repoName := strings.TrimSpace(string(repoOutput))

			// Create release notes
			releaseType := "Release"
			if prerelease {
				releaseType = "Development Release"
			}

			notes := fmt.Sprintf(`## %s

Automated build created on %s

### Binaries

Includes 26 binaries: 10 native, 8 WASM, 8 WASI

### Quick Start

Download and run a binary:
`+"```"+`bash
wget https://github.com/%s/releases/download/%s/decksh-darwin-arm64
chmod +x decksh-darwin-arm64
./decksh-darwin-arm64 --help
`+"```"+`

For WASM binaries, use with a WebAssembly runtime like wasmtime or wasmer.
`, releaseType, time.Now().Format("2006-01-02 15:04:05"), repoName, version)

			// Build gh release create command
			releaseArgs := []string{"release", "create", version}
			releaseArgs = append(releaseArgs, "--title", fmt.Sprintf("%s %s", releaseType, version))
			releaseArgs = append(releaseArgs, "--notes", notes)
			if prerelease {
				releaseArgs = append(releaseArgs, "--prerelease")
			}
			releaseArgs = append(releaseArgs, cfg.getDistGlob())

			fmt.Printf("Creating release %s...\n", version)
			releaseCmd := exec.CommandContext(ctx, "gh", releaseArgs...)
			releaseCmd.Stdout = os.Stdout
			releaseCmd.Stderr = os.Stderr
			if err := releaseCmd.Run(); err != nil {
				return fmt.Errorf("failed to create release: %w", err)
			}

			fmt.Printf("\n✓ Release %s created successfully!\n", version)
			fmt.Printf("View at: https://github.com/%s/releases/tag/%s\n", repoName, version)

			return nil
		},
	}

	cmd.Flags().BoolVar(&skipBuild, "skip-build", false, "Skip building and use existing dist/ binaries")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "Mark as prerelease (auto-enabled for dev-* versions)")
	cmd.Flags().StringVar(&version, "version", "", "Version tag (default: auto-generated timestamp)")

	return cmd
}

func newDevCleanCommand(cfg *config) *cobra.Command {
	return &cobra.Command{
		Use:   "dev-clean",
		Short: "Remove all dot folders (.data, .src, .dist, .fonts) for fresh start",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get all directories from config
			dirsToRemove := []string{
				dataDir,
				srcDir,
				distDir,
				fontsDir,
			}

			for _, dir := range dirsToRemove {
				if _, err := os.Stat(dir); err == nil {
					fmt.Printf("Removing %s...\n", dir)
					if err := os.RemoveAll(dir); err != nil {
						return fmt.Errorf("failed to remove %s: %w", dir, err)
					}
					fmt.Printf("✓ Removed %s\n", dir)
				} else {
					fmt.Printf("  Skipping %s (doesn't exist)\n", dir)
				}
			}

			fmt.Println("\n✓ Dev clean complete - all dot folders removed")
			return nil
		},
	}
}
