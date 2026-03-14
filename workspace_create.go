package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/x-motemen/ghq/cmdutil"
	"github.com/x-motemen/ghq/logger"
)

func doWorkspaceCreate(c *cli.Context) error {
	if c.NArg() == 0 {
		return fmt.Errorf("repo URL is required")
	}
	argURL := c.Args().First()

	profileName := c.String("profile")
	name := c.String("name")
	purpose := c.String("purpose")
	branch := c.String("branch")
	sparsePaths := c.StringSlice("sparse")

	// --sparse implies sparse profile
	if len(sparsePaths) > 0 {
		profileName = "sparse"
	}

	profile, ok := cloneProfiles[profileName]
	if !ok {
		return fmt.Errorf("unknown profile %q: must be one of task, readonly, full, sparse", profileName)
	}

	u, err := newURL(argURL, false, false)
	if err != nil {
		return fmt.Errorf("could not parse URL %q: %w", argURL, err)
	}

	repoShort := path.Base(strings.TrimSuffix(u.Path, ".git"))

	primaryRoot, err := primaryLocalRepositoryRoot()
	if err != nil {
		return err
	}
	workspaceRoot := filepath.Join(primaryRoot, "workspaces")

	if name == "" {
		name = repoShort + "-" + time.Now().UTC().Format("20060102-150405")
	}

	workspaceDir := filepath.Join(workspaceRoot, name)
	cloneDir := filepath.Join(workspaceDir, repoShort)

	if _, err := os.Stat(workspaceDir); err == nil {
		return fmt.Errorf("workspace %q already exists at %s", name, workspaceDir)
	}

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return err
	}

	// Resolve VCS backend — same pattern as getter.getRemoteRepository
	remote, err := NewRemoteRepository(u)
	if err != nil {
		os.RemoveAll(workspaceDir)
		return err
	}
	vcs, repoURL, err := remote.VCS()
	if err != nil {
		os.RemoveAll(workspaceDir)
		return err
	}

	logger.Log("workspace clone", fmt.Sprintf("%s -> %s", repoURL, cloneDir))

	opt := &vcsGetOption{
		url:    repoURL,
		dir:    cloneDir,
		silent: false,
	}
	applyProfile(opt, profile, branch)

	if err := vcs.Clone(opt); err != nil {
		os.RemoveAll(workspaceDir)
		return fmt.Errorf("clone failed: %w", err)
	}

	// Post-clone sparse checkout setup
	if profile.Sparse && len(sparsePaths) > 0 {
		if err := cmdutil.RunInDir(cloneDir, "git", "sparse-checkout", "init", "--cone"); err != nil {
			os.RemoveAll(workspaceDir)
			return fmt.Errorf("sparse-checkout init failed: %w", err)
		}
		args := append([]string{"sparse-checkout", "set"}, sparsePaths...)
		if err := cmdutil.RunInDir(cloneDir, "git", args...); err != nil {
			os.RemoveAll(workspaceDir)
			return fmt.Errorf("sparse-checkout set failed: %w", err)
		}
	}

	// Build CLAUDE.md
	mdData := claudeMDData{
		WorkspaceName: name,
		RepoURL:       repoURL.String(),
		Profile:       profileName,
		CloneFlags:    profileCloneFlags(profile, branch),
		Branch:        branch,
		SparsePaths:   sparsePaths,
		CreatedAt:     time.Now().UTC(),
		Purpose:       purpose,
	}
	if err := generateClaudeMD(workspaceDir, cloneDir, mdData); err != nil {
		// Non-fatal: workspace is usable without CLAUDE.md
		fmt.Fprintf(os.Stderr, "warning: could not generate CLAUDE.md: %v\n", err)
	}

	entry := WorkspaceEntry{
		Name:      name,
		Repo:      strings.TrimPrefix(repoURL.String(), repoURL.Scheme+"://"),
		Path:      workspaceDir,
		CloneDir:  cloneDir,
		Profile:   profileName,
		Branch:    branch,
		Purpose:   purpose,
		CreatedAt: mdData.CreatedAt,
	}
	if err := registryAdd(entry); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not update workspace registry: %v\n", err)
	}

	fmt.Fprintln(c.App.Writer, workspaceDir)
	return nil
}
