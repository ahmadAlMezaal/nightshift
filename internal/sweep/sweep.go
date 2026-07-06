// Package sweep runs autonomous maintenance sweeps (ENG-222): scan known repos, pick eligible catalog tasks (per-task cooldowns), run the agent, open a PR per change — same budget caps and worker pool as ticket work. Opt-in via SWEEP_ENABLED=true; off by default.
package sweep

import (
	"strings"
	"time"
)

// Task describes a maintenance task in the catalog.
type Task struct {
	Name string // unique identifier (e.g. "lint-cleanup")
	// Description is a short human-readable summary shown in logs and PR bodies.
	Description string
	// Cooldown is the minimum gap between runs of this task on the same repo — stops nightly re-runs.
	Cooldown time.Duration
	// Prompt returns the agent prompt; repoPath is the local checkout so prompts can reference it.
	Prompt func(repoPath string) string
	// BranchSuffix names the sweep branch ("lint-cleanup" → "noctra/sweep-<repo>-lint-cleanup"); must be unique across tasks.
	BranchSuffix string
	// CommitPrefix is the conventional-commit prefix (e.g. "chore", "fix").
	CommitPrefix string
	// PRLabel is an optional GitHub label applied to the PR (e.g. "maintenance").
	PRLabel string
}

// catalog is the built-in set of maintenance tasks, registered at init time.
var catalog []Task

// Register adds a task to the global catalog (called from task_*.go init funcs).
func Register(t Task) {
	catalog = append(catalog, t)
}

// Catalog returns a copy of all registered tasks.
func Catalog() []Task {
	out := make([]Task, len(catalog))
	copy(out, catalog)
	return out
}

// FilterTasks returns catalog tasks whose names are in enabled; nil/empty returns all.
func FilterTasks(enabled []string) []Task {
	if len(enabled) == 0 {
		return Catalog()
	}
	set := make(map[string]bool, len(enabled))
	for _, n := range enabled {
		set[n] = true
	}
	var out []Task
	for _, t := range catalog {
		if set[t.Name] {
			out = append(out, t)
		}
	}
	return out
}

// SweepBranchName returns the sweep branch (e.g. "noctra/sweep-myrepo-lint-cleanup") — distinct from ticket "noctra/<identifier>"; embeds the repo slug so the watcher can reconstruct the identifier.
func SweepBranchName(repoSlug, taskSuffix string) string {
	return "noctra/" + strings.ToLower(SweepIdentifier(repoSlug, taskSuffix))
}

// SweepIdentifier returns the worktree identifier (e.g. "SWEEP-MYREPO-LINT-CLEANUP") — the worktree dir name and active-set dedup key.
func SweepIdentifier(repoSlug, taskSuffix string) string {
	return strings.ToUpper("SWEEP-" + sanitizeRepoSlug(repoSlug) + "-" + taskSuffix)
}

func sanitizeRepoSlug(repoSlug string) string {
	return strings.ReplaceAll(repoSlug, "/", "-")
}

// ParseSweepIdentifier reverses SweepIdentifier ("SWEEP-MYREPO-LINT-CLEANUP" → "myrepo", "lint-cleanup");
// ok is false for non-sweep identifiers. Both halves contain dashes, so the split is anchored on the known
// task suffixes — longest match wins so a repo slug ending in a task word can't mis-split.
func ParseSweepIdentifier(identifier string) (repoSlug, taskSuffix string, ok bool) {
	rest, found := strings.CutPrefix(strings.ToLower(identifier), "sweep-")
	if !found {
		return "", "", false
	}
	best := ""
	for _, t := range catalog {
		if s := t.BranchSuffix; strings.HasSuffix(rest, "-"+s) && len(s) > len(best) {
			best = s
		}
	}
	if best == "" {
		return "", "", false
	}
	return strings.TrimSuffix(rest, "-"+best), best, true
}
