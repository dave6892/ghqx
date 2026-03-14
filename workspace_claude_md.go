package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type claudeMDData struct {
	WorkspaceName string
	RepoURL       string
	Profile       string
	CloneFlags    string
	Branch        string
	SparsePaths   []string
	CreatedAt     time.Time
	Purpose       string
}

// generateClaudeMD writes a CLAUDE.md to workspaceDir by merging three sources:
// 1. ghq-generated metadata header
// 2. User template: ~/.config/ghq/workspace-template.md (optional)
// 3. Project template: <cloneDir>/.ghq/workspace-template.md (optional)
func generateClaudeMD(workspaceDir, cloneDir string, data claudeMDData) error {
	var b strings.Builder

	// 1. ghq-generated header
	b.WriteString(fmt.Sprintf("# Workspace: %s\n\n", data.WorkspaceName))
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	b.WriteString(fmt.Sprintf("| Repo | %s |\n", data.RepoURL))
	b.WriteString(fmt.Sprintf("| Profile | %s |\n", data.Profile))
	b.WriteString(fmt.Sprintf("| Clone flags | `%s` |\n", data.CloneFlags))
	if data.Branch != "" {
		b.WriteString(fmt.Sprintf("| Branch | %s |\n", data.Branch))
	}
	if len(data.SparsePaths) > 0 {
		b.WriteString(fmt.Sprintf("| Sparse paths | %s |\n", strings.Join(data.SparsePaths, ", ")))
	}
	b.WriteString(fmt.Sprintf("| Created | %s |\n", data.CreatedAt.UTC().Format(time.RFC3339)))
	if data.Purpose != "" {
		b.WriteString(fmt.Sprintf("| Purpose | %s |\n", data.Purpose))
	}
	b.WriteString("\n---\n")

	// 2. User template
	if content := readOptionalTemplate(userWorkspaceTemplatePath()); content != "" {
		b.WriteString("\n")
		b.WriteString(content)
	}

	// 3. Project template
	if content := readOptionalTemplate(filepath.Join(cloneDir, ".ghq", "workspace-template.md")); content != "" {
		b.WriteString("\n")
		b.WriteString(content)
	}

	return os.WriteFile(filepath.Join(workspaceDir, "CLAUDE.md"), []byte(b.String()), 0644)
}

func userWorkspaceTemplatePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "ghq", "workspace-template.md")
}

func readOptionalTemplate(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
