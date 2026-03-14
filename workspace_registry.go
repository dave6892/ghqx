package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// WorkspaceEntry is one record in the workspace registry.
type WorkspaceEntry struct {
	Name      string    `json:"name"`
	Repo      string    `json:"repo"`
	Path      string    `json:"path"`
	CloneDir  string    `json:"clone_dir"`
	Profile   string    `json:"profile"`
	Branch    string    `json:"branch,omitempty"`
	Purpose   string    `json:"purpose,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func workspaceRegistryPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ghq", "workspaces.json"), nil
}

func loadRegistry() ([]WorkspaceEntry, error) {
	p, err := workspaceRegistryPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return []WorkspaceEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []WorkspaceEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func saveRegistry(entries []WorkspaceEntry) error {
	p, err := workspaceRegistryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func registryAdd(entry WorkspaceEntry) error {
	entries, err := loadRegistry()
	if err != nil {
		return err
	}
	entries = append(entries, entry)
	return saveRegistry(entries)
}

func registryRemove(name string) error {
	entries, err := loadRegistry()
	if err != nil {
		return err
	}
	filtered := entries[:0]
	found := false
	for _, e := range entries {
		if e.Name == name {
			found = true
		} else {
			filtered = append(filtered, e)
		}
	}
	if !found {
		return nil
	}
	return saveRegistry(filtered)
}

func registryFind(name string) (*WorkspaceEntry, error) {
	entries, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].Name == name {
			return &entries[i], nil
		}
	}
	return nil, nil
}
