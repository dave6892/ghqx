package main

// CloneProfile maps a named profile to git clone behaviour.
type CloneProfile struct {
	Shallow bool
	Partial string // "blobless" | "treeless" | ""
	NoTags  bool
	Sparse  bool
}

// cloneProfiles are the built-in named profiles for workspace clones.
var cloneProfiles = map[string]CloneProfile{
	"task":     {Shallow: true, Partial: "blobless"},
	"readonly": {Shallow: true, Partial: "blobless", NoTags: true},
	"full":     {},
	"sparse":   {Shallow: true, Partial: "blobless", Sparse: true},
}

// applyProfile sets vcsGetOption fields from a CloneProfile and branch.
// A non-empty branch triggers --single-branch inside GitBackend.Clone.
func applyProfile(opt *vcsGetOption, p CloneProfile, branch string) {
	opt.shallow = p.Shallow
	opt.partial = p.Partial
	opt.branch = branch
	if p.NoTags {
		opt.ExtraArgs = append(opt.ExtraArgs, "--no-tags")
	}
}

// profileCloneFlags returns a human-readable summary of flags a profile applies.
func profileCloneFlags(p CloneProfile, branch string) string {
	flags := "--depth 1"
	if !p.Shallow {
		flags = ""
	}
	if branch != "" {
		flags += " --branch " + branch + " --single-branch"
	}
	if p.Partial == "blobless" {
		flags += " --filter=blob:none"
	} else if p.Partial == "treeless" {
		flags += " --filter=tree:0"
	}
	if p.NoTags {
		flags += " --no-tags"
	}
	if flags == "" {
		flags = "(none)"
	}
	return flags
}
