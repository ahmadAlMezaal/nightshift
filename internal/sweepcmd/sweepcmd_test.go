package sweepcmd

import "testing"

func TestParseArgs(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		want    request
		help    bool
		wantErr bool
	}{
		{name: "bare", args: nil},
		{name: "now is a no-op", args: []string{"--now"}},
		{name: "force", args: []string{"--force"}, want: request{Force: true}},
		{name: "short force", args: []string{"-f"}, want: request{Force: true}},
		{
			name: "task and repo",
			args: []string{"--task", "lint-cleanup", "--repo", "trade-mate"},
			want: request{Tasks: []string{"lint-cleanup"}, Repos: []string{"trade-mate"}},
		},
		{
			name: "repeatable",
			args: []string{"-t", "a", "-t", "b", "-r", "x"},
			want: request{Tasks: []string{"a", "b"}, Repos: []string{"x"}},
		},
		{name: "help", args: []string{"--help"}, help: true},
		{name: "unknown flag", args: []string{"--nope"}, wantErr: true},
		{name: "task without value", args: []string{"--task"}, wantErr: true},
		{name: "repo without value", args: []string{"--repo"}, wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, help, err := parseArgs(c.args)
			if c.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if help != c.help {
				t.Fatalf("help: got %v, want %v", help, c.help)
			}
			if got.Force != c.want.Force {
				t.Errorf("force: got %v, want %v", got.Force, c.want.Force)
			}
			if len(got.Tasks) != len(c.want.Tasks) {
				t.Fatalf("tasks: got %v, want %v", got.Tasks, c.want.Tasks)
			}
			for i := range got.Tasks {
				if got.Tasks[i] != c.want.Tasks[i] {
					t.Errorf("tasks[%d]: got %q, want %q", i, got.Tasks[i], c.want.Tasks[i])
				}
			}
			if len(got.Repos) != len(c.want.Repos) {
				t.Fatalf("repos: got %v, want %v", got.Repos, c.want.Repos)
			}
		})
	}
}

// TestDescribe covers the confirmation line users read back before the sweep starts.
func TestDescribe(t *testing.T) {
	if got := describe(request{}); got != "all eligible tasks" {
		t.Errorf("bare: got %q", got)
	}
	if got := describe(request{Force: true}); got != "all eligible tasks; cooldowns ignored" {
		t.Errorf("force: got %q", got)
	}
	got := describe(request{Tasks: []string{"lint-cleanup"}, Repos: []string{"trade-mate"}})
	if got != "tasks: lint-cleanup; repos: trade-mate" {
		t.Errorf("filters: got %q", got)
	}
}
