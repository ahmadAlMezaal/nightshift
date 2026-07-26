package linear

import "testing"

func TestUpsertRepoDirective(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		ref, branch string
		want        string
	}{
		{
			name: "empty content gets the directive",
			ref:  "acme/app",
			want: "Repo: acme/app\n",
		},
		{
			name:    "prepended above existing prose",
			content: "Some project notes.\n",
			ref:     "acme/app",
			want:    "Repo: acme/app\n\nSome project notes.\n",
		},
		{
			name:    "existing directive replaced in place",
			content: "Intro\n\nRepo: acme/old\n\nOutro\n",
			ref:     "acme/new",
			want:    "Intro\n\nRepo: acme/new\n\nOutro\n",
		},
		{
			name:    "branch written next to the repo",
			content: "Repo: acme/old\n",
			ref:     "acme/app",
			branch:  "staging",
			want:    "Repo: acme/app\nBranch: staging\n",
		},
		{
			name:    "empty branch drops a stale pin",
			content: "Repo: acme/old\nBranch: staging\n",
			ref:     "acme/app",
			want:    "Repo: acme/app\n",
		},
		{
			name:    "duplicate directives collapse to one",
			content: "Repo: a/one\nRepo: b/two\n",
			ref:     "c/three",
			want:    "Repo: c/three\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpsertRepoDirective(tt.content, tt.ref, tt.branch)
			if got != tt.want {
				t.Errorf("UpsertRepoDirective() = %q, want %q", got, tt.want)
			}

			p := Project{Content: got}
			gotRef, gotBranch := p.RepoDirective()
			if gotRef != tt.ref {
				t.Errorf("round-trip ref = %q, want %q", gotRef, tt.ref)
			}
			if gotBranch != tt.branch {
				t.Errorf("round-trip branch = %q, want %q", gotBranch, tt.branch)
			}
		})
	}
}

func TestMatchProjects(t *testing.T) {
	projects := []Project{
		{Name: "Noctra", SlugID: "abc123", URL: "https://linear.app/acme/project/noctra-abc123"},
		{Name: "Noctra Site", SlugID: "def456", URL: "https://linear.app/acme/project/noctra-site-def456"},
		{Name: "Sandbox", SlugID: "ghi789", URL: "https://linear.app/acme/project/sandbox-ghi789"},
	}

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"exact name wins over substring", "Noctra", []string{"Noctra"}},
		{"exact name is case-insensitive", "noctra site", []string{"Noctra Site"}},
		{"substring returns every candidate", "noct", []string{"Noctra", "Noctra Site"}},
		{"url resolves by slug id", "https://linear.app/acme/project/noctra-site-def456", []string{"Noctra Site"}},
		{"url with trailing path still resolves", "https://linear.app/acme/project/sandbox-ghi789/overview", []string{"Sandbox"}},
		{"unknown url matches nothing", "https://linear.app/acme/project/other-zzz999", nil},
		{"unknown name matches nothing", "nothing-like-this", nil},
		{"blank matches nothing", "  ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchProjects(projects, tt.query)
			if len(got) != len(tt.want) {
				t.Fatalf("MatchProjects(%q) returned %d projects, want %d", tt.query, len(got), len(tt.want))
			}
			for i, name := range tt.want {
				if got[i].Name != name {
					t.Errorf("match %d = %q, want %q", i, got[i].Name, name)
				}
			}
		})
	}
}
