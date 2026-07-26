package reposcmd

import "testing"

func TestParseAddArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    addOptions
		wantErr bool
	}{
		{
			name: "no args prompts for everything",
			args: nil,
		},
		{
			name: "positional repo",
			args: []string{"acme/app"},
			want: addOptions{Ref: "acme/app"},
		},
		{
			name: "separated flag values",
			args: []string{"acme/app", "--project", "Noctra", "--branch", "staging"},
			want: addOptions{Ref: "acme/app", Project: "Noctra", Branch: "staging"},
		},
		{
			name: "equals form",
			args: []string{"--project=Noctra", "--branch=staging", "acme/app"},
			want: addOptions{Ref: "acme/app", Project: "Noctra", Branch: "staging"},
		},
		{
			name: "short flags",
			args: []string{"-p", "Noctra", "-b", "main"},
			want: addOptions{Project: "Noctra", Branch: "main"},
		},
		{
			name:    "flag without a value",
			args:    []string{"--project"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"--nope"},
			wantErr: true,
		},
		{
			name:    "two positional repos",
			args:    []string{"acme/app", "acme/other"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAddArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseAddArgs(%v) succeeded, want an error", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAddArgs(%v): %v", tt.args, err)
			}
			if got != tt.want {
				t.Errorf("parseAddArgs(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestIsSkip(t *testing.T) {
	for _, answer := range []string{"skip", "SKIP", " none ", "no"} {
		if !isSkip(answer) {
			t.Errorf("isSkip(%q) = false, want true", answer)
		}
	}
	for _, answer := range []string{"", "Noctra", "nope"} {
		if isSkip(answer) {
			t.Errorf("isSkip(%q) = true, want false", answer)
		}
	}
}
