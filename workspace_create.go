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

	// urfave/cli/v2 stops flag parsing at the first non-flag argument, so
	// flags placed after the URL (e.g. `create <url> --name foo`) land in
	// c.Args() instead of being parsed. We scan all args to extract them.
	argURL, extraFlags := parseWorkspaceCreateArgs(c.Args().Slice())
	if argURL == "" {
		return fmt.Errorf("repo URL is required")
	}

	profileName := firstNonEmpty(c.String("profile"), extraFlags["profile"], "task")
	name := firstNonEmpty(c.String("name"), extraFlags["name"])
	purpose := firstNonEmpty(c.String("purpose"), extraFlags["purpose"])
	branch := firstNonEmpty(c.String("branch"), extraFlags["branch"])
	sparsePaths := c.StringSlice("sparse")
	if len(sparsePaths) == 0 {
		if s := extraFlags["sparse"]; s != "" {
			sparsePaths = strings.Split(s, ",")
		}
	}

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
		name = repoShort + "-" + time.Now().Format("20060102-150405")
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

// parseWorkspaceCreateArgs separates the repo URL from any flags that ended
// up in the args slice because they were placed after the first positional arg
// (urfave/cli/v2 stops flag parsing at the first non-flag argument).
// Returns the URL and a map of flag-name → value for the known string flags.
func parseWorkspaceCreateArgs(args []string) (url string, flags map[string]string) {
	flags = make(map[string]string)
	knownFlags := map[string]bool{
		"name": true, "profile": true, "purpose": true, "branch": true, "b": true, "sparse": true,
	}
	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if knownFlags[key] && i+1 < len(args) {
				flags[key] = args[i+1]
				i += 2
				continue
			}
		} else if strings.HasPrefix(arg, "-") {
			key := strings.TrimPrefix(arg, "-")
			if knownFlags[key] && i+1 < len(args) {
				flags[key] = args[i+1]
				i += 2
				continue
			}
		} else if url == "" {
			url = arg
		}
		i++
	}
	// Normalize -b alias
	if v, ok := flags["b"]; ok && flags["branch"] == "" {
		flags["branch"] = v
	}
	return url, flags
}

// firstNonEmpty returns the first non-empty string from the given values.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
