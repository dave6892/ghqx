package main

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// wsHelpTemplate returns a CustomHelpTemplate for a workspace subcommand
// that correctly shows "ghq workspace <name> <args>" in the USAGE line.
// urfave/cli/v2 never populates commandNamePath, so .FullName() always
// returns just .Name — we hardcode the parent path here instead.
func wsHelpTemplate(argsUsage string) string {
	return fmt.Sprintf(`NAME:
    {{.Name}} - {{.Usage}}

USAGE:
    ghq workspace {{.Name}} %s
{{if (len .Description)}}
DESCRIPTION: {{.Description}}
{{end}}{{if (len .Flags)}}
OPTIONS:
    {{range .Flags}}{{.}}
    {{end}}
{{end}}`, argsUsage)
}

var commandWorkspace = &cli.Command{
	Name:    "workspace",
	Aliases: []string{"ws"},
	Usage:   "Manage agent workspaces",
	Subcommands: []*cli.Command{
		commandWorkspaceCreate,
		commandWorkspaceList,
		commandWorkspaceRemove,
		commandWorkspaceClean,
		commandWorkspaceLook,
		commandWorkspaceStatus,
		commandWorkspaceUpdate,
		commandWorkspaceRoot,
	},
}

var commandWorkspaceCreate = &cli.Command{
	Name:               "create",
	ArgsUsage:          "<repo-url>",
	Usage:              "Clone a repo into an isolated agent workspace",
	Action:             doWorkspaceCreate,
	CustomHelpTemplate: wsHelpTemplate("[--profile <profile>] [--name <name>] [--purpose <text>] [--branch <branch>] [--sparse <path>] <repo-url>"),
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "profile",
			Value: "task",
			Usage: "Clone profile: task (default), readonly, full, sparse",
		},
		&cli.StringFlag{
			Name:  "name",
			Usage: "Workspace name (default: <repo>-<timestamp>)",
		},
		&cli.StringFlag{
			Name:  "purpose",
			Usage: "Human-readable description written to CLAUDE.md",
		},
		&cli.StringFlag{
			Name:    "branch",
			Aliases: []string{"b"},
			Usage:   "Branch to clone",
		},
		&cli.StringSliceFlag{
			Name:  "sparse",
			Usage: "Sparse checkout path(s), implies --profile sparse",
		},
	},
}

var commandWorkspaceList = &cli.Command{
	Name:               "list",
	Usage:              "List all workspaces",
	Action:             doWorkspaceList,
	CustomHelpTemplate: wsHelpTemplate("[--json]"),
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "Output as JSON"},
	},
}

var commandWorkspaceRemove = &cli.Command{
	Name:               "remove",
	Aliases:            []string{"rm"},
	ArgsUsage:          "<name> [<name>...]",
	Usage:              "Delete a workspace and its registry entry",
	Action:             doWorkspaceRemove,
	CustomHelpTemplate: wsHelpTemplate("[--dry-run] <name> [<name>...]"),
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be removed without removing"},
	},
}

var commandWorkspaceClean = &cli.Command{
	Name:               "clean",
	ArgsUsage:          "[pattern]",
	Usage:              "Bulk remove workspaces by age or name pattern",
	Action:             doWorkspaceClean,
	CustomHelpTemplate: wsHelpTemplate("[--older-than <duration>] [--dry-run] [<pattern>]"),
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "older-than",
			Usage: "Remove workspaces older than this duration (e.g. 7d, 24h)",
		},
		&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be removed without removing"},
	},
}

var commandWorkspaceLook = &cli.Command{
	Name:               "look",
	ArgsUsage:          "<name>",
	Usage:              "Open a shell in a workspace directory",
	Action:             doWorkspaceLook,
	CustomHelpTemplate: wsHelpTemplate("<name>"),
}

var commandWorkspaceStatus = &cli.Command{
	Name:               "status",
	ArgsUsage:          "[name]",
	Usage:              "Show git status of one or all workspaces",
	Action:             doWorkspaceStatus,
	CustomHelpTemplate: wsHelpTemplate("[<name>]"),
}

var commandWorkspaceUpdate = &cli.Command{
	Name:               "update",
	ArgsUsage:          "<name>",
	Usage:              "Pull latest changes in a workspace",
	Action:             doWorkspaceUpdate,
	CustomHelpTemplate: wsHelpTemplate("<name>"),
}

var commandWorkspaceRoot = &cli.Command{
	Name:               "root",
	Usage:              "Print the workspace root directory",
	Action:             doWorkspaceRoot,
	CustomHelpTemplate: wsHelpTemplate(""),
}
