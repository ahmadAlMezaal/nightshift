package notify

import "testing"

func TestEscapeMarkdown(t *testing.T) {
	cases := []struct{ in, want string }{
		{"snake_case_thing", `snake\_case\_thing`},
		{"*emphasis*", `\*emphasis\*`},
		{"`code`", "\\`code\\`"},
		{"[link]", `\[link]`},
		{"feat: implement ENG-42 — add_user.go", `feat: implement ENG-42 — add\_user.go`},
		{`already\_escaped`, `already\\_escaped`},
		{"Nothing special here", "Nothing special here"},
		{"", ""},
	}
	for _, c := range cases {
		if got := EscapeMarkdown(c.in); got != c.want {
			t.Errorf("EscapeMarkdown(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
