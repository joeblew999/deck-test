package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// Constants
// =============================================================================

// Directory structure constants
const (
	dataDir = ".data"
	srcDir  = ".src"
	distDir = ".dist"
)

// =============================================================================
// Types
// =============================================================================

type buildTarget string

const (
	targetNative buildTarget = "native"
	targetWASM   buildTarget = "wasm"
	targetWASI   buildTarget = "wasi"
)

type binSpec struct {
	name        string
	pkg         string // Go import path
	repo        string // which repo provides this
	wasmSupport bool
	wasiSupport bool
	requiresUI  bool
}

type buildResult struct {
	binary string
	target buildTarget
	path   string
	err    error
}

type repoConfig struct {
	name      string
	url       string
	dir       string
	branch    string
	depth     int
	filterRaw string
	filter    []string
	sparseRaw string
	sparse    []string
	isData    bool
}

type config struct {
	goCmd        string
	gitCmd       string
	goBinDir     string
	distDir      string // absolute path to dist directory
	deckfontsEnv string
	repos        map[string]*repoConfig
	toolchain    []binSpec
	skipEnsure   bool
}

// =============================================================================
// Build Target Methods
// =============================================================================

func (t buildTarget) buildEnv() (goos, goarch string) {
	switch t {
	case targetWASM:
		return "js", "wasm"
	case targetWASI:
		return "wasip1", "wasm"
	default:
		return "", ""
	}
}

func (t buildTarget) extension() string {
	if t == targetWASM || t == targetWASI {
		return ".wasm"
	}
	return ""
}

// =============================================================================
// Config Loading and Initialization
// =============================================================================

func loadConfig() (*config, error) {
	cfg := &config{
		goCmd:  getenvDefault("GO", "go"),
		gitCmd: getenvDefault("GIT", "git"),
		repos:  make(map[string]*repoConfig),
	}

	// Initialize repositories and toolchain
	cfg.initDataRepos()
	cfg.initCodeRepos()
	cfg.initToolchain()

	// Resolve go bin directory
	binDir, err := resolveGoBin(cfg.goCmd)
	if err != nil {
		return nil, err
	}
	cfg.goBinDir = binDir

	return cfg, nil
}

func (cfg *config) initDataRepos() {
	cfg.addDataRepo("deckviz", "deckviz", "master")
	cfg.addDataRepo("deckfonts", "deckfonts", "master")

	dubois := cfg.addDataRepo("dubois", "dubois-data-portraits", "master")
	dubois.filterRaw = getenvDefault("DUBOIS_FILTER", "--filter=blob:none")
	dubois.sparseRaw = os.Getenv("DUBOIS_SPARSE")
}

func (cfg *config) initCodeRepos() {
	cfg.addCodeRepo("deck", "master")
	cfg.addCodeRepo("decksh", "master")
	cfg.addCodeRepo("ebcanvas", "main")
	cfg.addCodeRepo("giocanvas", "master")
	cfg.addCodeRepo("giftsh", "main")
	cfg.addCodeRepo("gift", "master")
}

func (cfg *config) initToolchain() {
	cfg.toolchain = []binSpec{
		// decksh tools
		{name: "decksh", pkg: "github.com/ajstarks/decksh/cmd/decksh", repo: "decksh", wasmSupport: true, wasiSupport: true},
		{name: "dshfmt", pkg: "github.com/ajstarks/decksh/cmd/dshfmt", repo: "decksh", wasmSupport: true, wasiSupport: true},
		{name: "dshlint", pkg: "github.com/ajstarks/decksh/cmd/dshlint", repo: "decksh", wasmSupport: true, wasiSupport: true},

		// deck tools
		{name: "pdfdeck", pkg: "github.com/ajstarks/deck/cmd/pdfdeck", repo: "deck", wasmSupport: true, wasiSupport: true},
		{name: "pngdeck", pkg: "github.com/ajstarks/deck/cmd/pngdeck", repo: "deck", wasmSupport: true, wasiSupport: true},
		{name: "svgdeck", pkg: "github.com/ajstarks/deck/cmd/svgdeck", repo: "deck", wasmSupport: true, wasiSupport: true},

		// gift tools
		{name: "gift", pkg: "github.com/ajstarks/gift", repo: "gift", wasmSupport: true, wasiSupport: true},
		{name: "giftsh", pkg: "github.com/ajstarks/giftsh", repo: "giftsh", wasmSupport: true, wasiSupport: true},

		// UI apps (native only)
		{name: "ebdeck", pkg: "github.com/ajstarks/ebcanvas/ebdeck", repo: "ebcanvas", requiresUI: true},
		{name: "gcdeck", pkg: "github.com/ajstarks/giocanvas/gcdeck", repo: "giocanvas", requiresUI: true},
	}
}

func (cfg *config) addDataRepo(name, dir, branch string) *repoConfig {
	repo := &repoConfig{
		name:   name,
		url:    getenvDefault(strings.ToUpper(name)+"_REPO", fmt.Sprintf("https://github.com/ajstarks/%s.git", dir)),
		dir:    getenvDefault(strings.ToUpper(name)+"_DIR", filepath.Join(dataDir, dir)),
		branch: getenvDefault(strings.ToUpper(name)+"_BRANCH", branch),
		depth:  getenvInt(strings.ToUpper(name)+"_DEPTH", 1),
		isData: true,
	}
	cfg.repos[name] = repo
	return repo
}

func (cfg *config) addCodeRepo(name, branch string) *repoConfig {
	repo := &repoConfig{
		name:   name,
		url:    getenvDefault(strings.ToUpper(name)+"_REPO", fmt.Sprintf("https://github.com/ajstarks/%s.git", name)),
		dir:    getenvDefault(strings.ToUpper(name)+"_DIR", filepath.Join(srcDir, name)),
		branch: getenvDefault(strings.ToUpper(name)+"_BRANCH", branch),
		depth:  getenvInt(strings.ToUpper(name)+"_DEPTH", 1),
		isData: false,
	}
	cfg.repos[name] = repo
	return repo
}

func (cfg *config) finalize() error {
	// Resolve all repo directories to absolute paths
	for _, repo := range cfg.repos {
		var err error
		if repo.dir, err = absPath(repo.dir); err != nil {
			return fmt.Errorf("resolve %s dir: %w", repo.name, err)
		}
		repo.filter = strings.Fields(strings.TrimSpace(repo.filterRaw))
		repo.sparse = strings.Fields(strings.TrimSpace(repo.sparseRaw))
	}

	// Resolve dist directory to absolute path
	var err error
	if cfg.distDir, err = absPath(distDir); err != nil {
		return fmt.Errorf("resolve dist dir: %w", err)
	}

	// Set DECKFONTS environment variable
	envFonts := strings.TrimSpace(os.Getenv("DECKFONTS"))
	if envFonts == "" {
		envFonts = cfg.repos["deckfonts"].dir
	}
	if cfg.deckfontsEnv, err = absPath(envFonts); err != nil {
		return err
	}

	return nil
}

// =============================================================================
// Path Helpers (all file paths calculated here)
// =============================================================================

// getBinaryPath returns the path to a platform-specific binary in the dist directory
func (cfg *config) getBinaryPath(name string) string {
	return filepath.Join(cfg.distDir, name+"-"+runtime.GOOS+"-"+runtime.GOARCH)
}

// getDistGlob returns the glob pattern for all files in dist directory (for gh release)
func (cfg *config) getDistGlob() string {
	return filepath.Join(filepath.Base(cfg.distDir), "*")
}

// getGoBinPath returns the path to a binary in the go bin directory
func (cfg *config) getGoBinPath(name string) string {
	return filepath.Join(cfg.goBinDir, name)
}

// getExampleDir returns the directory path for an example
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

// getExampleDshPath returns the path to an example's .dsh file
func (cfg *config) getExampleDshPath(dir, name string) string {
	return filepath.Join(dir, name+".dsh")
}

// getExampleXmlPath returns the path to an example's .xml output file
func (cfg *config) getExampleXmlPath(dir, name string) string {
	return filepath.Join(dir, name+".xml")
}

// getShellCompletionPath returns the default path for shell completion file
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

// getShellRCPath returns the path to shell RC file
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

// resolveBinary resolves the path to a binary by searching in order:
// 1. dist directory (our built/downloaded binaries)
// 2. PATH (system-installed binaries)
// 3. GOBIN directory (go-installed binaries)
func (cfg *config) resolveBinary(name string) (string, error) {
	// Check dist/ directory first (our downloaded binaries)
	distPath := cfg.getBinaryPath(name)
	if _, err := os.Stat(distPath); err == nil {
		return distPath, nil
	}

	// Fallback to PATH
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}

	// Finally check goBinDir
	path := cfg.getGoBinPath(name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("%s not found in %s, PATH, or %s", name, cfg.distDir, cfg.goBinDir)
}

// =============================================================================
// Repository Management
// =============================================================================

func (cfg *config) ensureRepos(ctx context.Context) error {
	for _, repo := range cfg.repos {
		if repo.isData {
			if err := cfg.gitCloneOrUpdate(ctx, repo); err != nil {
				return err
			}
		}
	}
	return nil
}

func (cfg *config) ensureBuildRepos(ctx context.Context) error {
	for _, repo := range cfg.repos {
		if !repo.isData {
			if err := cfg.gitCloneOrUpdate(ctx, repo); err != nil {
				return err
			}
		}
	}
	return nil
}

func (cfg *config) gitCloneOrUpdate(ctx context.Context, repo *repoConfig) error {
	if _, err := os.Stat(filepath.Join(repo.dir, ".git")); err == nil {
		return cfg.gitUpdate(ctx, repo)
	}
	return cfg.gitClone(ctx, repo)
}

func (cfg *config) gitClone(ctx context.Context, repo *repoConfig) error {
	args := []string{"clone"}
	if repo.depth > 0 {
		args = append(args, fmt.Sprintf("--depth=%d", repo.depth))
	}
	args = append(args, repo.filter...)
	args = append(args, "--branch", repo.branch, repo.url, repo.dir)

	fmt.Printf("Cloning %s into %s\n", repo.url, repo.dir)
	if err := cfg.runGit(ctx, args...); err != nil {
		return err
	}

	// Handle sparse checkout if configured
	if len(repo.sparse) > 0 {
		if err := cfg.runGit(ctx, "-C", repo.dir, "sparse-checkout", "init", "--cone"); err != nil {
			return err
		}
		setArgs := append([]string{"-C", repo.dir, "sparse-checkout", "set"}, repo.sparse...)
		if err := cfg.runGit(ctx, setArgs...); err != nil {
			return err
		}
	}
	return nil
}

func (cfg *config) gitUpdate(ctx context.Context, repo *repoConfig) error {
	args := []string{"-C", repo.dir, "fetch"}
	if repo.depth > 0 {
		args = append(args, fmt.Sprintf("--depth=%d", repo.depth))
	}
	args = append(args, repo.filter...)
	args = append(args, "origin", repo.branch)

	fmt.Printf("Updating %s\n", repo.dir)
	if err := cfg.runGit(ctx, args...); err != nil {
		return err
	}
	if err := cfg.runGit(ctx, "-C", repo.dir, "checkout", repo.branch); err != nil {
		return err
	}
	if err := cfg.runGit(ctx, "-C", repo.dir, "reset", "--hard", "origin/"+repo.branch); err != nil {
		return err
	}

	// Update sparse checkout if configured
	if len(repo.sparse) > 0 {
		setArgs := append([]string{"-C", repo.dir, "sparse-checkout", "set"}, repo.sparse...)
		if err := cfg.runGit(ctx, setArgs...); err != nil {
			return err
		}
	}
	return nil
}

func (cfg *config) runGit(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, cfg.gitCmd, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// =============================================================================
// Workspace Management
// =============================================================================

func (cfg *config) ensureWorkspace(ctx context.Context) error {
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("create %s dir: %w", srcDir, err)
	}

	workFile := filepath.Join(srcDir, "go.work")

	// Build workspace content
	var dirs []string
	dirs = append(dirs, "..") // Parent directory (deck-test)

	for _, repo := range cfg.repos {
		if !repo.isData {
			repoName := filepath.Base(repo.dir)
			dirs = append(dirs, "./"+repoName)
		}
	}

	// Create go.work file
	content := "go 1.25\n\n"
	for _, dir := range dirs {
		content += fmt.Sprintf("use %s\n", dir)
	}

	if err := os.WriteFile(workFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write go.work: %w", err)
	}

	fmt.Printf("✓ Created %s/go.work workspace file\n", srcDir)
	return nil
}

// =============================================================================
// Binary Management
// =============================================================================

func (cfg *config) ensureBins(ctx context.Context) error {
	// Download from GitHub releases only
	return cfg.downloadReleaseBinaries(ctx)
}

func (cfg *config) downloadReleaseBinaries(ctx context.Context) error {
	// Check if gh CLI is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found. Install it with: brew install gh")
	}

	// Get latest release info
	fmt.Println("Checking for latest release...")
	listCmd := exec.CommandContext(ctx, "gh", "release", "list", "--limit", "1")
	output, err := listCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	// Parse release tag from output (format: "TITLE\tTYPE\tTAG\tDATE" - tab separated)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("no releases found")
	}
	// Split by tab to get fields
	fields := strings.Split(lines[0], "\t")
	if len(fields) < 3 {
		return fmt.Errorf("failed to parse release info")
	}
	releaseTag := strings.TrimSpace(fields[2]) // TAG is the 3rd field
	fmt.Printf("Latest release: %s\n", releaseTag)

	// Get release published time
	viewCmd := exec.CommandContext(ctx, "gh", "release", "view", releaseTag, "--json", "publishedAt", "-q", ".publishedAt")
	timeOutput, err := viewCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get release time: %w", err)
	}
	releaseTime, err := time.Parse(time.RFC3339, strings.TrimSpace(string(timeOutput)))
	if err != nil {
		return fmt.Errorf("failed to parse release time: %w", err)
	}

	// Create dist directory if it doesn't exist
	if err := os.MkdirAll(cfg.distDir, 0755); err != nil {
		return fmt.Errorf("create dist dir: %w", err)
	}

	// Download native binaries for current platform
	downloaded := 0
	skipped := 0
	for _, spec := range cfg.toolchain {
		filename := cfg.buildFilename(spec.name, targetNative)
		destPath := filepath.Join(cfg.distDir, filename)

		// Check if local binary exists and compare timestamps
		fileInfo, err := os.Stat(destPath)
		if err == nil {
			// File exists - check if it's newer than the release
			localModTime := fileInfo.ModTime()
			if localModTime.After(releaseTime) {
				fmt.Printf("✓ %s is up to date (local is newer)\n", filename)
				skipped++
				continue
			}
			fmt.Printf("⟳ %s needs update (release is newer)\n", filename)
		}

		fmt.Printf("Downloading %s...\n", filename)
		downloadCmd := exec.CommandContext(ctx, "gh", "release", "download", releaseTag, "-p", filename, "-D", cfg.distDir, "--clobber")
		downloadCmd.Stdout = os.Stdout
		downloadCmd.Stderr = os.Stderr
		if err := downloadCmd.Run(); err != nil {
			fmt.Printf("⚠ Failed to download %s: %v\n", filename, err)
			continue
		}

		// Make executable
		if err := os.Chmod(destPath, 0755); err != nil {
			fmt.Printf("⚠ Failed to chmod %s: %v\n", filename, err)
		}

		downloaded++
		fmt.Printf("✓ Downloaded %s\n", filename)
	}

	if downloaded == 0 && skipped > 0 {
		fmt.Println("All binaries are up to date")
	} else if downloaded > 0 {
		fmt.Printf("✓ Downloaded %d binaries (%d up to date)\n", downloaded, skipped)
	}

	return nil
}

// =============================================================================
// Build Functions
// =============================================================================

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

// =============================================================================
// Utility Functions
// =============================================================================

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

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err = io.Copy(destination, source); err != nil {
		return err
	}

	// Preserve executable permissions
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}
