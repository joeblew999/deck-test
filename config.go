package main

import (
	"fmt"
	"strings"
)

// =============================================================================
// Constants
// =============================================================================

// Directory structure constants
const (
	dataDir  = ".data"
	srcDir   = ".src"
	distDir  = ".dist"
	fontsDir = ".fonts"
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
	goCmd     string
	gitCmd    string
	goBinDir  string
	distDir   string // absolute path to dist directory
	fontsDir  string // absolute path to fonts directory
	repos     map[string]*repoConfig
	fontsRepo *repoConfig // deckfonts repo (managed separately)
	toolchain []binSpec
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
	if err := cfg.initFontsRepo(); err != nil {
		return nil, err
	}

	// Resolve go bin directory
	binDir, err := resolveGoBin(cfg.goCmd)
	if err != nil {
		return nil, err
	}
	cfg.goBinDir = binDir

	return cfg, nil
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

	// Resolve fonts repo directory to absolute path
	if cfg.fontsRepo.dir, err = absPath(cfg.fontsRepo.dir); err != nil {
		return fmt.Errorf("resolve fonts dir: %w", err)
	}
	cfg.fontsDir = cfg.fontsRepo.dir

	return nil
}
