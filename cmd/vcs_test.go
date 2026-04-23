package cmd

import "testing"

func TestTrimBookmark(t *testing.T) {
	cases := map[string]string{
		"":                "",
		"main":            "main",
		"main*":           "main",
		"main@origin":     "main",
		"main*@origin":    "main",
		"feature/x":       "feature/x",
		"feature/x*":      "feature/x",
		"feature/x@fork":  "feature/x",
		"feature/x*@fork": "feature/x",
	}
	for in, want := range cases {
		if got := trimBookmark(in); got != want {
			t.Errorf("trimBookmark(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseJJRemotes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want map[string]string
	}{
		{"empty", "", map[string]string{}},
		{
			"single origin",
			"origin https://github.com/user/repo.git",
			map[string]string{"origin": "https://github.com/user/repo.git"},
		},
		{
			"origin and upstream",
			"origin https://github.com/user/repo.git\nupstream https://github.com/other/repo.git",
			map[string]string{
				"origin":   "https://github.com/user/repo.git",
				"upstream": "https://github.com/other/repo.git",
			},
		},
		{
			"trailing newline",
			"origin https://github.com/user/repo.git\n",
			map[string]string{"origin": "https://github.com/user/repo.git"},
		},
		{
			"malformed line skipped",
			"garbage\norigin https://example.com/r.git",
			map[string]string{"origin": "https://example.com/r.git"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseJJRemotes(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("len=%d want %d: got=%v", len(got), len(c.want), got)
			}
			for k, v := range c.want {
				if got[k] != v {
					t.Errorf("remote %q = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}
