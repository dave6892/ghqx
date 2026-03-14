# ghqx Usage Guide

ghqx is a fork of [ghq](https://github.com/x-motemen/ghq) that extends the original with additional commands like `workspace` and `migrate`. It organizes remote repository clones under a structured directory layout based on the repository URL.

```
~/ghq
├── github.com/
│   ├── user/
│   │   └── repo/
│   └── org/
│       └── project/
├── gitlab.com/
│   └── team/
│       └── service/
└── workspaces/          ← isolated agent/task clones (ghqx extension)
    ├── fix-auth-bug/
    └── review-pr-42/
```

## Installation

### Build from source (recommended for now)

Requires [Go 1.21+](https://go.dev/dl/).

```zsh
# Clone the repo
ghq get github.com/dave6892/ghqx   # if you already have ghq/ghqx
# or
git clone https://github.com/dave6892/ghqx.git ~/ghqx && cd ~/ghqx

# Install to $GOPATH/bin (make sure $GOPATH/bin is in your $PATH)
go install -ldflags="-s -w" .

# Verify
ghqx --version
```

Or use `make install` if you have the dev dependencies:

```zsh
make install
```

### Shell setup

Add an alias or shell function to make navigation easier:

```zsh
# ~/.zshrc

# cd into a repo selected with fzf
function gcd() {
  local repo
  repo=$(ghqx list -p | fzf) && cd "$repo"
}

# Optional: alias ghq to ghqx if replacing ghq entirely
# alias ghq=ghqx
```

### git config

Set your ghq root (defaults to `~/ghq` if not set):

```zsh
git config --global ghq.root ~/ghq
```

---

## Quick Start

```zsh
# Clone a repo
ghqx get github.com/user/repo

# List all local repos (pipe to fzf for interactive navigation)
cd "$(ghqx list -p | fzf)"

# Create an isolated workspace for an agent task
ghqx ws create github.com/user/repo --name fix-login --purpose "Fix login bug"

# See all workspaces
ghqx ws list

# Clean up when done
ghqx ws remove fix-login
```

## Configuration

ghqx uses `git-config` variables for configuration.

- **`ghq.root`** — Root directory for cloned repositories (default: `~/ghq`). Supports multiple values; the last one is the primary root.
- **`ghq.user`** — Default owner name when only a project name is given (defaults to `$USER`).
- **`ghq.completeUser`** — Set to `false` to use the project name as the owner (e.g. `ghq get vim` → `github.com/vim/vim`).
- **`ghq.<url>.vcs`** — Explicitly set the VCS backend for a URL pattern.
- **`ghq.<url>.root`** — Set a repository-specific root directory.
- **`GHQ_ROOT`** env var — Overrides all `ghq.root` settings.

## Commands

### `ghqx get`

Clone a remote repository into the ghq root directory.

```zsh path=null start=null
# Clone by full URL
ghqx get https://github.com/user/repo

# Clone by shorthand (uses github.com by default)
ghqx get user/repo

# Clone via SSH
ghqx get -p user/repo

# Update an already-cloned repo
ghqx get -u user/repo

# Shallow clone
ghqx get --shallow user/repo

# Clone a specific branch
ghqx get -b main user/repo

# Bare clone
ghqx get --bare user/repo

# Partial clone (blobless or treeless)
ghqx get --partial blobless user/repo

# Clone multiple repos in parallel
ghqx get -P user/repo1 user/repo2

# Clone and open a shell in the repo directory
ghqx get -l user/repo
```

**Aliases:** `ghqx clone`

### `ghqx list`

List locally cloned repositories.

```zsh path=null start=null
# List all repos (relative paths)
ghqx list

# Filter by query
ghqx list myproject

# Exact match
ghqx list -e user/repo

# Full paths
ghqx list -p

# Unique shortest subpaths
ghqx list --unique

# Filter by VCS backend
ghqx list --vcs git
```

**Tip:** Combine with tools like `fzf` for interactive selection:

```zsh path=null start=null
cd "$(ghqx list -p | fzf)"
```

### `ghqx create`

Create a new local repository under the ghq root.

```zsh path=null start=null
ghqx create user/new-repo
ghqx create --vcs git user/new-repo
ghqx create --bare user/new-repo
```

### `ghqx rm`

Remove a local repository.

```zsh path=null start=null
# Preview what would be removed
ghqx rm --dry-run user/repo

# Remove (prompts for confirmation — type y to confirm)
ghqx rm user/repo
```

### `ghqx root`

Print the ghq root directory.

```zsh path=null start=null
# Primary root
ghqx root

# All configured roots
ghqx root --all
```

### `ghqx migrate`

Migrate an existing repository directory into the ghq-managed structure. It detects the VCS backend, reads the remote URL, and moves the repository to the correct location.

```zsh path=null start=null
# Preview the migration
ghqx migrate --dry-run ~/projects/my-repo

# Migrate (prompts for confirmation)
ghqx migrate ~/projects/my-repo

# Skip confirmation
ghqx migrate -y ~/projects/my-repo
```

If the repository has linked git worktrees, `ghqx migrate` automatically runs `git worktree repair` after moving.

---

## Workspace Command

The `workspace` command (`ws` for short) manages isolated, disposable clones designed for agent-driven or task-scoped workflows. Workspaces live under `<ghq-root>/workspaces/` and are tracked in a JSON registry at `~/.config/ghq/workspaces.json`.

### `ghqx workspace create`

Clone a repo into a new isolated workspace.

```zsh path=null start=null
# Basic — creates a workspace named <repo>-<timestamp>
ghqx ws create https://github.com/user/repo

# Custom name
ghqx ws create https://github.com/user/repo --name my-feature

# With a description (written to CLAUDE.md)
ghqx ws create https://github.com/user/repo --purpose "Fix login bug"

# Specific branch
ghqx ws create https://github.com/user/repo -b feature-branch

# Sparse checkout (only clone specific paths)
ghqx ws create https://github.com/user/repo --sparse src/api --sparse docs
```

The command prints the workspace directory path on success.

#### Clone Profiles

The `--profile` flag controls how the repo is cloned:

| Profile    | Behavior |
|------------|----------|
| `task`     | Default. Shallow clone + blobless partial clone (`--depth 1 --filter=blob:none`). |
| `readonly` | Like `task` but also skips tags (`--no-tags`). |
| `full`     | Full clone with complete history. |
| `sparse`   | Like `task` plus sparse checkout. Automatically set when `--sparse` is used. |

#### CLAUDE.md Generation

Each workspace gets a `CLAUDE.md` file at its root containing metadata (name, repo, profile, branch, timestamps). You can extend this with templates:

- **User template:** `~/.config/ghq/workspace-template.md` — appended to every workspace.
- **Project template:** `<repo>/.ghq/workspace-template.md` — appended for that specific repo.

### `ghqx workspace list`

List all registered workspaces.

```zsh path=null start=null
# Table output
ghqx ws list

# JSON output
ghqx ws list --json
```

### `ghqx workspace status`

Show git status (uncommitted changes, unpushed commits) for workspaces.

```zsh path=null start=null
# All workspaces
ghqx ws status

# Specific workspace
ghqx ws status my-feature
```

### `ghqx workspace look`

Open a shell inside a workspace directory. Sets `GHQ_WORKSPACE` in the environment.

```zsh path=null start=null
ghqx ws look my-feature
```

### `ghqx workspace update`

Pull latest changes in a workspace.

```zsh path=null start=null
ghqx ws update my-feature
```

### `ghqx workspace remove`

Delete one or more workspaces (directory + registry entry).

```zsh path=null start=null
# Preview
ghqx ws remove --dry-run my-feature

# Remove
ghqx ws remove my-feature

# Remove multiple
ghqx ws remove ws-1 ws-2 ws-3
```

**Aliases:** `ghqx ws rm`

### `ghqx workspace clean`

Bulk remove workspaces by age or name pattern.

```zsh path=null start=null
# Remove workspaces older than 7 days
ghqx ws clean --older-than 7d

# Remove workspaces matching a pattern
ghqx ws clean "experiment"

# Combine both (OR logic)
ghqx ws clean "experiment" --older-than 24h

# Preview
ghqx ws clean --older-than 30d --dry-run
```

Supported duration formats: standard Go durations (`24h`, `1h30m`) and a day shorthand (`7d`).

### `ghqx workspace root`

Print the workspace root directory.

```zsh path=null start=null
ghqx ws root
# e.g. /Users/you/ghq/workspaces
```

---

## Agentic Workflow

Workspaces are designed for scenarios where one or more coding agents need isolated, disposable copies of a repository — running in parallel without interfering with each other or with your canonical clone.

### Typical lifecycle

```zsh path=null start=null
# 1. Create a workspace for a specific task
ghqx ws create git@github.com:org/repo.git \
  --name agent-fix-payments \
  --purpose "Investigate and fix payment timeout errors" \
  --profile task

# 2. The workspace is created at ~/ghq/workspaces/agent-fix-payments/
#    with a CLAUDE.md describing the task.
#    The agent reads CLAUDE.md automatically on entry.

# 3. Check what workspaces are active
ghqx ws list

# 4. Check if an agent left uncommitted work before cleaning up
ghqx ws status

# 5. Remove specific workspaces when done
ghqx ws remove agent-fix-payments

# 6. Or bulk-clean old workspaces (e.g. after a sprint)
ghqx ws clean --older-than 7d --dry-run   # preview first
ghqx ws clean --older-than 7d
```

### Parallel agents on the same repo

```zsh path=null start=null
ghqx ws create git@github.com:org/repo.git --name agent-1 --purpose "Feature A"
ghqx ws create git@github.com:org/repo.git --name agent-2 --purpose "Feature B"
ghqx ws create git@github.com:org/repo.git --name agent-3 --purpose "Code review"

ghqx ws list
# NAME      PROFILE  REPO           CREATED
# agent-1   task     org/repo       2026-03-13
# agent-2   task     org/repo       2026-03-13
# agent-3   readonly org/repo       2026-03-13

ghqx ws remove agent-1 agent-2 agent-3
```

### Customising CLAUDE.md

Every workspace gets a `CLAUDE.md` at its root. Content is merged from three sources in order:

1. **ghq metadata** (always generated) — repo, profile, clone flags, branch, created date, purpose
2. **User template** at `~/.config/ghq/workspace-template.md` — your standing instructions for all agents
3. **Project template** at `<repo>/.ghq/workspace-template.md` — repo-specific agent instructions committed by the project

Example user template (`~/.config/ghq/workspace-template.md`):

```markdown
## Agent Instructions

- Run `git status` before starting any work.
- Keep commits small and well-described.
- Do not push directly to main.
- If you are unsure, stop and ask rather than guessing.
```

Example project template (`.ghq/workspace-template.md` committed in the repo):

```markdown
## Project-Specific Notes

- Run `make test` before committing.
- API keys are in `.env.example` — copy to `.env` and fill in values.
- The payments module requires a running Postgres instance (see `docker-compose.yml`).
```

### Choosing a profile

| Task type | Recommended profile | Why |
|-----------|-------------------|-----|
| General feature work | `task` (default) | Shallow + blobless — fast clone, writable |
| Code review / analysis | `readonly` | Same as task but skips tags — clearly non-write intent |
| Debugging with `git log` / `git blame` | `full` | Needs complete history |
| Monorepo — agent only touches one subdirectory | `sparse` | Only materialises the paths you specify |
