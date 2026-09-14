package cmd

import "testing"

func TestVersion(t *testing.T) {
	if v := Version(); v == "" {
		t.Fatal("Version() returned empty string")
	}
}

func TestFormatRevision(t *testing.T) {
	cases := []struct {
		name     string
		rev      string
		modified bool
		want     string
	}{
		{"truncates to 12", "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0", false, "a1b2c3d4e5f6"},
		{"modified appends +dirty", "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0", true, "a1b2c3d4e5f6+dirty"},
		{"empty revision yields devel", "", false, "devel"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatRevision(c.rev, c.modified); got != c.want {
				t.Errorf("formatRevision(%q, %v) = %q, want %q", c.rev, c.modified, got, c.want)
			}
		})
	}
}
