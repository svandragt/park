package cmd

import "testing"

func TestSshToHTTPS(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"git@github.com:svandragt/park.git", "https://github.com/svandragt/park"},
		{"git@github.com:svandragt/park", "https://github.com/svandragt/park"},
		{"https://github.com/svandragt/park", "https://github.com/svandragt/park"},
		{"https://github.com/svandragt/park.git", "https://github.com/svandragt/park"},
		{"", ""},
	}
	for _, c := range cases {
		got := sshToHTTPS(c.in)
		if got != c.want {
			t.Errorf("sshToHTTPS(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeRemote(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"git@github.com:svandragt/park.git", "https://github.com/svandragt/park"},
		{"git@github.com:svandragt/park", "https://github.com/svandragt/park"},
		{"https://github.com/svandragt/park", "https://github.com/svandragt/park"},
		{"https://github.com/svandragt/park.git", "https://github.com/svandragt/park"},
		{"https://github.com/svandragt/park/", "https://github.com/svandragt/park"},
		{"", ""},
	}
	for _, c := range cases {
		got := normalizeRemote(c.in)
		if got != c.want {
			t.Errorf("normalizeRemote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
