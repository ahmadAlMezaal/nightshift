package repo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

// TestCreateAndCleanupWorktree drives real git against a temp repo.
func TestCreateAndCleanupWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repo := t.TempDir()
	mustGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}

	mustGit("init", "-b", "main", "--quiet")
	mustGit("config", "user.email", "t@t")
	mustGit("config", "user.name", "T")
	// Throwaway fixture — keep commits hermetic regardless of host git config.
	mustGit("config", "commit.gpgsign", "false")
	mustGit("remote", "add", "origin", repo)

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("init"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustGit("add", "-A")
	mustGit("commit", "-m", "init", "--quiet")
	mustGit("fetch", "origin", "--quiet")

	base := t.TempDir()
	ctx := context.Background()

	wt, err := CreateWorktree(ctx, base, "ENG-200", repo, "main")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if wt.Branch != "noctra/eng-200" {
		t.Errorf("branch: got %q, want %q", wt.Branch, "noctra/eng-200")
	}
	if _, err := os.Stat(wt.Path); err != nil {
		t.Fatalf("worktree dir missing: %v", err)
	}

	CleanupWorktree(ctx, repo, base, "ENG-200")
	if _, err := os.Stat(wt.Path); !os.IsNotExist(err) {
		t.Errorf("worktree dir should have been removed (err=%v)", err)
	}
}

func TestCreateWorktree_BadRepoPath(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	_, err := CreateWorktree(context.Background(), t.TempDir(), "ENG-999", "/does/not/exist", "main")
	if err == nil {
		t.Fatal("expected CreateWorktree to fail on a bad repo path")
	}
}

// TestResumeWorktree_PicksUpExistingBranchCommits: after committing+pushing a marker on the branch, ResumeWorktree carries it forward instead of recreating from main.
func TestResumeWorktree_PicksUpExistingBranchCommits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repo := t.TempDir()
	mustGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}

	mustGit("init", "-b", "main", "--quiet")
	mustGit("config", "user.email", "t@t")
	mustGit("config", "user.name", "T")
	mustGit("config", "commit.gpgsign", "false")
	mustGit("config", "receive.denyCurrentBranch", "ignore")
	mustGit("remote", "add", "origin", repo)

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("init"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustGit("add", "-A")
	mustGit("commit", "-m", "init", "--quiet")
	mustGit("fetch", "origin", "--quiet")

	base := t.TempDir()
	ctx := context.Background()

	// Round 1 — fresh ticket: worktree from main, commit, push.
	wt1, err := CreateWorktree(ctx, base, "ENG-300", repo, "main")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	markerPath := filepath.Join(wt1.Path, "from-attempt-1.txt")
	if err := os.WriteFile(markerPath, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	runInWt := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = wt1.Path
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v in worktree: %v\n%s", args, err, string(out))
		}
	}
	runInWt("add", "-A")
	runInWt("commit", "-m", "attempt-1", "--quiet")
	runInWt("push", "-u", "origin", wt1.Branch, "--quiet")

	// Tear down between rounds, as in a real lifecycle (cleanup after the first PR).
	CleanupWorktree(ctx, repo, base, "ENG-300")

	// Round 2 — resume: should bring back the marker, NOT start fresh.
	wt2, err := ResumeWorktree(ctx, base, "ENG-300", repo)
	if err != nil {
		t.Fatalf("ResumeWorktree: %v", err)
	}
	if wt2.Branch != "noctra/eng-300" {
		t.Errorf("branch: got %q", wt2.Branch)
	}
	if _, err := os.Stat(filepath.Join(wt2.Path, "from-attempt-1.txt")); err != nil {
		t.Errorf("resumed worktree is missing the prior attempt's marker file: %v", err)
	}

	CleanupWorktree(ctx, repo, base, "ENG-300")
}

// TestResumeWorktree_OverStaleWorktree resumes while a stale worktree for the branch is still checked out (e.g. a crash skipped cleanup): cleanup must remove the worktree before the branch, else `worktree add -b` fails.
func TestResumeWorktree_OverStaleWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repo := t.TempDir()
	mustGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}

	mustGit("init", "-b", "main", "--quiet")
	mustGit("config", "user.email", "t@t")
	mustGit("config", "user.name", "T")
	mustGit("config", "commit.gpgsign", "false")
	mustGit("config", "receive.denyCurrentBranch", "ignore")
	mustGit("remote", "add", "origin", repo)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("init"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustGit("add", "-A")
	mustGit("commit", "-m", "init", "--quiet")
	mustGit("fetch", "origin", "--quiet")

	base := t.TempDir()
	ctx := context.Background()

	wt1, err := CreateWorktree(ctx, base, "ENG-400", repo, "main")
	if err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(wt1.Path, "marker.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	runInWt := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = wt1.Path
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v in worktree: %v\n%s", args, err, string(out))
		}
	}
	runInWt("add", "-A")
	runInWt("commit", "-m", "attempt-1", "--quiet")
	runInWt("push", "-u", "origin", wt1.Branch, "--quiet")

	// Deliberately DO NOT clean up — the branch stays checked out in wt1.
	wt2, err := ResumeWorktree(ctx, base, "ENG-400", repo)
	if err != nil {
		t.Fatalf("ResumeWorktree over a stale worktree: %v", err)
	}
	if _, err := os.Stat(filepath.Join(wt2.Path, "marker.txt")); err != nil {
		t.Errorf("resumed worktree missing prior marker: %v", err)
	}

	CleanupWorktree(ctx, repo, base, "ENG-400")
}

func TestResumeWorktree_FailsIfBranchNotOnRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repo := t.TempDir()
	mustGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}

	mustGit("init", "-b", "main", "--quiet")
	mustGit("config", "user.email", "t@t")
	mustGit("config", "user.name", "T")
	mustGit("config", "commit.gpgsign", "false")
	mustGit("remote", "add", "origin", repo)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("init"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustGit("add", "-A")
	mustGit("commit", "-m", "init", "--quiet")
	mustGit("fetch", "origin", "--quiet")

	if _, err := ResumeWorktree(context.Background(), t.TempDir(), "ENG-NEVER-PUSHED", repo); err == nil {
		t.Fatal("expected ResumeWorktree to fail when the branch isn't on origin")
	}
}

// newFixtureRepo builds a throwaway clone with one commit on main and an origin remote pointing at itself.
func newFixtureRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, string(out))
		}
	}
	mustGit("init", "-b", "main", "--quiet")
	mustGit("config", "user.email", "t@t")
	mustGit("config", "user.name", "T")
	mustGit("config", "commit.gpgsign", "false")
	mustGit("remote", "add", "origin", dir)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("init"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustGit("add", "-A")
	mustGit("commit", "-m", "init", "--quiet")
	mustGit("fetch", "origin", "--quiet")
	return dir
}

// TestCreateWorktreeWithBranch_ConcurrentSameRepo is the regression test for two sweep tasks landing on
// one repo in a single cycle: both drive `git worktree add` against the shared clone, and without the
// per-repo lock the loser dies on "could not lock config file .git/config".
func TestCreateWorktreeWithBranch_ConcurrentSameRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repoDir := newFixtureRepo(t)
	base := t.TempDir()
	ctx := context.Background()

	const n = 8
	errs := make(chan error, n)
	var start sync.WaitGroup
	start.Add(1)
	for i := 0; i < n; i++ {
		go func(i int) {
			start.Wait() // release all goroutines at once to maximise contention
			id := fmt.Sprintf("SWEEP-FIXTURE-TASK-%d", i)
			_, err := CreateWorktreeWithBranch(ctx, base, id, repoDir, "main", "noctra/sweep-task-"+strconv.Itoa(i))
			errs <- err
		}(i)
	}
	start.Done()

	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent CreateWorktreeWithBranch failed: %v", err)
		}
	}
}

// TestLockRepo_DistinctReposDoNotBlock guards the lock's granularity: it must be per clone, not global,
// or sweeps across different repos would serialize needlessly.
func TestLockRepo_DistinctReposDoNotBlock(t *testing.T) {
	unlockA := lockRepo("/repos/a")
	defer unlockA()

	done := make(chan struct{})
	go func() {
		lockRepo("/repos/b")()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("lockRepo serialized two distinct repos")
	}
}

// TestLockRepo_SamePathSerializes confirms differing spellings of one path share a lock.
func TestLockRepo_SamePathSerializes(t *testing.T) {
	unlock := lockRepo("/repos/a")

	acquired := make(chan struct{})
	go func() {
		lockRepo("/repos/a/../a")() // same clone, different spelling
		close(acquired)
	}()

	select {
	case <-acquired:
		t.Fatal("lockRepo let two holders into the same repo")
	case <-time.After(100 * time.Millisecond):
	}

	unlock()
	select {
	case <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("lockRepo did not release")
	}
}
