package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/x-motemen/ghq/cmdutil"
)

func doWorkspaceList(c *cli.Context) error {
	entries, err := loadRegistry()
	if err != nil {
		return err
	}
	if c.Bool("json") {
		data, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(c.App.Writer, string(data))
		return nil
	}
	w := tabwriter.NewWriter(c.App.Writer, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPROFILE\tREPO\tCREATED")
	for _, e := range entries {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			e.Name, e.Profile, e.Repo,
			e.CreatedAt.UTC().Format("2006-01-02"))
	}
	return w.Flush()
}

func doWorkspaceRemove(c *cli.Context) error {
	if c.NArg() == 0 {
		return fmt.Errorf("at least one workspace name is required")
	}
	dryRun := c.Bool("dry-run")
	var errs []string
	for _, name := range c.Args().Slice() {
		entry, err := registryFind(name)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if entry == nil {
			errs = append(errs, fmt.Sprintf("%s: not found", name))
			continue
		}
		if dryRun {
			fmt.Fprintf(c.App.Writer, "would remove: %s\n", entry.Path)
			continue
		}
		if err := os.RemoveAll(entry.Path); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if err := registryRemove(name); err != nil {
			errs = append(errs, fmt.Sprintf("%s: registry: %v", name, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}

func doWorkspaceClean(c *cli.Context) error {
	entries, err := loadRegistry()
	if err != nil {
		return err
	}

	pattern := c.Args().First()
	olderThanStr := c.String("older-than")
	dryRun := c.Bool("dry-run")

	var maxAge time.Duration
	if olderThanStr != "" {
		maxAge, err = parseDuration(olderThanStr)
		if err != nil {
			return fmt.Errorf("invalid --older-than value: %w", err)
		}
	}

	var keep []WorkspaceEntry
	for _, e := range entries {
		remove := false
		if olderThanStr != "" && time.Since(e.CreatedAt) > maxAge {
			remove = true
		}
		if pattern != "" && strings.Contains(e.Name, pattern) {
			remove = true
		}
		if !remove {
			keep = append(keep, e)
			continue
		}
		if dryRun {
			fmt.Fprintf(c.App.Writer, "would remove: %s (%s)\n", e.Name, e.Path)
		} else {
			if err := os.RemoveAll(e.Path); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not remove %s: %v\n", e.Path, err)
				keep = append(keep, e) // keep entry if removal failed
			} else {
				fmt.Fprintf(c.App.Writer, "removed: %s\n", e.Name)
			}
		}
	}
	if !dryRun {
		return saveRegistry(keep)
	}
	return nil
}

func doWorkspaceLook(c *cli.Context) error {
	if c.NArg() == 0 {
		return fmt.Errorf("workspace name is required")
	}
	name := c.Args().First()
	entry, err := registryFind(name)
	if err != nil {
		return err
	}
	if entry == nil {
		return fmt.Errorf("workspace %q not found", name)
	}
	cmd := exec.Command(detectShell())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = entry.Path
	cmd.Env = append(os.Environ(), "GHQ_WORKSPACE="+entry.Name)
	return cmdutil.RunCommand(cmd, true)
}

func doWorkspaceStatus(c *cli.Context) error {
	entries, err := loadRegistry()
	if err != nil {
		return err
	}
	nameFilter := c.Args().First()

	w := tabwriter.NewWriter(c.App.Writer, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tREPO\tSTATUS")
	for _, e := range entries {
		if nameFilter != "" && e.Name != nameFilter {
			continue
		}
		status := workspaceGitStatus(e.CloneDir)
		fmt.Fprintf(w, "%s\t%s\t%s\n", e.Name, e.Repo, status)
	}
	return w.Flush()
}

func workspaceGitStatus(cloneDir string) string {
	// Uncommitted changes
	out, err := runGitOutput(cloneDir, "status", "--porcelain")
	uncommitted := 0
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if strings.TrimSpace(line) != "" {
				uncommitted++
			}
		}
	}

	// Unpushed commits (ignore error — no upstream is normal for shallow clones)
	unpushed := 0
	out2, err2 := runGitOutput(cloneDir, "log", "@{u}..HEAD", "--oneline")
	if err2 == nil {
		for _, line := range strings.Split(strings.TrimSpace(out2), "\n") {
			if strings.TrimSpace(line) != "" {
				unpushed++
			}
		}
	}

	if uncommitted == 0 && unpushed == 0 {
		return "clean"
	}
	var parts []string
	if uncommitted > 0 {
		parts = append(parts, fmt.Sprintf("%d uncommitted", uncommitted))
	}
	if unpushed > 0 {
		parts = append(parts, fmt.Sprintf("%d unpushed", unpushed))
	}
	return strings.Join(parts, ", ")
}

func runGitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}

func doWorkspaceUpdate(c *cli.Context) error {
	if c.NArg() == 0 {
		return fmt.Errorf("workspace name is required")
	}
	name := c.Args().First()
	entry, err := registryFind(name)
	if err != nil {
		return err
	}
	if entry == nil {
		return fmt.Errorf("workspace %q not found", name)
	}
	u, err := newURL(entry.Repo, false, false)
	if err != nil {
		return fmt.Errorf("could not parse repo URL %q: %w", entry.Repo, err)
	}
	return GitBackend.Update(&vcsGetOption{
		url:    u,
		dir:    entry.CloneDir,
		silent: false,
	})
}

func doWorkspaceRoot(c *cli.Context) error {
	primary, err := primaryLocalRepositoryRoot()
	if err != nil {
		return err
	}
	fmt.Fprintln(c.App.Writer, filepath.Join(primary, "workspaces"))
	return nil
}

func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
