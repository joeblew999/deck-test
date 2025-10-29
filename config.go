package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type binSpec struct {
	name string
	pkg  string
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
}

type config struct {
	goCmd  string
	gitCmd string

	goBinDir     string
	deckfontsEnv string

	deckviz   repoConfig
	deckfonts repoConfig
	dubois    repoConfig

	toolchain []binSpec

	skipEnsure bool
}

func loadConfig() (*config, error) {
	cfg := &config{
		goCmd:  getenvDefault("GO", "go"),
		gitCmd: getenvDefault("GIT", "git"),
		toolchain: []binSpec{
			{name: "decksh", pkg: "github.com/ajstarks/decksh/cmd/decksh"},
			{name: "dshfmt", pkg: "github.com/ajstarks/decksh/cmd/dshfmt"},
			{name: "dshlint", pkg: "github.com/ajstarks/decksh/cmd/dshlint"},
			{name: "ebdeck", pkg: "github.com/ajstarks/ebcanvas/ebdeck"},
			{name: "pdfdeck", pkg: "github.com/ajstarks/deck/cmd/pdfdeck"},
			{name: "pngdeck", pkg: "github.com/ajstarks/deck/cmd/pngdeck"},
			{name: "svgdeck", pkg: "github.com/ajstarks/deck/cmd/svgdeck"},
			{name: "giftsh", pkg: "github.com/ajstarks/giftsh"},
			{name: "gift", pkg: "github.com/ajstarks/gift"},
			{name: "gcdeck", pkg: "github.com/ajstarks/giocanvas/gcdeck"},
		},
		deckviz: repoConfig{
			name:   "deckviz",
			url:    getenvDefault("DECKVIZ_REPO", "https://github.com/ajstarks/deckviz.git"),
			dir:    getenvDefault("DECKVIZ_DIR", "deckviz"),
			branch: getenvDefault("DECKVIZ_BRANCH", "master"),
			depth:  getenvInt("DECKVIZ_DEPTH", 1),
		},
		deckfonts: repoConfig{
			name:   "deckfonts",
			url:    getenvDefault("DECKFONTS_REPO", "https://github.com/ajstarks/deckfonts.git"),
			dir:    getenvDefault("DECKFONTS_DIR", "deckfonts"),
			branch: getenvDefault("DECKFONTS_BRANCH", "master"),
			depth:  getenvInt("DECKFONTS_DEPTH", 1),
		},
		dubois: repoConfig{
			name:      "dubois-data-portraits",
			url:       getenvDefault("DUBOIS_REPO", "https://github.com/ajstarks/dubois-data-portraits.git"),
			dir:       getenvDefault("DUBOIS_DIR", "dubois-data-portraits"),
			branch:    getenvDefault("DUBOIS_BRANCH", "master"),
			depth:     getenvInt("DUBOIS_DEPTH", 1),
			filterRaw: getenvDefault("DUBOIS_FILTER", "--filter=blob:none"),
			sparseRaw: os.Getenv("DUBOIS_SPARSE"),
		},
	}

	binDir, err := resolveGoBin(cfg.goCmd)
	if err != nil {
		return nil, err
	}
	cfg.goBinDir = binDir

	return cfg, nil
}

func (cfg *config) finalize() error {
	var err error

	if cfg.deckviz.dir, err = absPath(cfg.deckviz.dir); err != nil {
		return err
	}
	if cfg.deckfonts.dir, err = absPath(cfg.deckfonts.dir); err != nil {
		return err
	}
	if cfg.dubois.dir, err = absPath(cfg.dubois.dir); err != nil {
		return err
	}

	envFonts := strings.TrimSpace(os.Getenv("DECKFONTS"))
	if envFonts == "" {
		envFonts = cfg.deckfonts.dir
	}
	if cfg.deckfontsEnv, err = absPath(envFonts); err != nil {
		return err
	}

	cfg.dubois.filter = strings.Fields(strings.TrimSpace(cfg.dubois.filterRaw))
	cfg.dubois.sparse = strings.Fields(strings.TrimSpace(cfg.dubois.sparseRaw))

	return nil
}

func (cfg *config) ensureBins(ctx context.Context) error {
	env := append(os.Environ(), "GOBIN="+cfg.goBinDir)
	for _, spec := range cfg.toolchain {
		if _, err := cfg.resolveBinary(spec.name); err == nil {
			continue
		}
		fmt.Printf("Installing %s...\n", spec.name)
		cmd := exec.CommandContext(ctx, cfg.goCmd, "install", spec.pkg+"@latest")
		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("install %s: %w", spec.name, err)
		}
	}
	return nil
}

func (cfg *config) ensureRepos(ctx context.Context) error {
	if err := cfg.gitCloneOrUpdate(ctx, &cfg.deckviz); err != nil {
		return err
	}
	if err := cfg.gitCloneOrUpdate(ctx, &cfg.deckfonts); err != nil {
		return err
	}
	if err := cfg.gitCloneOrUpdate(ctx, &cfg.dubois); err != nil {
		return err
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
