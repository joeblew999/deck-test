package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Repository management functions

func (cfg *config) initDataRepos() {
	cfg.addDataRepo("deckviz", "deckviz", "master")

	dubois := cfg.addDataRepo("dubois", "dubois-data-portraits", "master")
	dubois.filterRaw = getenvDefault("DUBOIS_FILTER", "--filter=blob:none")
	dubois.sparseRaw = os.Getenv("DUBOIS_SPARSE")
}

func (cfg *config) initFontsRepo() error {
	// Create fonts repo config (clone to .fonts directory)
	cfg.fontsRepo = &repoConfig{
		name:   "deckfonts",
		url:    getenvDefault("DECKFONTS_REPO", "https://github.com/ajstarks/deckfonts.git"),
		dir:    getenvDefault("DECKFONTS_DIR", fontsDir),
		branch: getenvDefault("DECKFONTS_BRANCH", "master"),
		depth:  getenvInt("DECKFONTS_DEPTH", 1),
		isData: true,
	}
	cfg.fontsDir = cfg.fontsRepo.dir // Store dir path (will be made absolute in finalize())

	// Add to repos map so it gets cloned by ensureRepos()
	cfg.repos["deckfonts"] = cfg.fontsRepo

	return nil
}

func (cfg *config) initCodeRepos() {
	cfg.addCodeRepo("deck", "master")
	cfg.addCodeRepo("decksh", "master")
	cfg.addCodeRepo("ebcanvas", "main")
	cfg.addCodeRepo("giocanvas", "master")
	cfg.addCodeRepo("giftsh", "main")
	cfg.addCodeRepo("gift", "master")
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
