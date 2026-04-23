package cmd

import "strings"

// currentRemote returns the origin URL, trying git then jj.
func currentRemote() string {
	if r := gitOutput("remote", "get-url", "origin"); r != "" {
		return r
	}
	if url, ok := parseJJRemotes(jjOutput("git", "remote", "list"))["origin"]; ok {
		return url
	}
	return ""
}

// currentBranch returns the current branch or jj bookmark, trying git then jj.
// For jj, returns the nearest bookmark reachable from @ walking ancestors.
func currentBranch() string {
	if b := gitOutput("branch", "--show-current"); b != "" {
		return b
	}
	out := jjOutput("log", "-r", "heads(::@ & bookmarks())",
		"-T", `bookmarks.join(",") ++ "\n"`, "--no-graph", "--limit", "1")
	out = strings.TrimSpace(out)
	if out == "" {
		return ""
	}
	// If multiple bookmarks share the commit, take the first.
	if i := strings.Index(out, ","); i >= 0 {
		out = out[:i]
	}
	return trimBookmark(out)
}

// trimBookmark strips jj's trailing markers from a bookmark name:
// "*" (diverged), "@remote" (remote-tracking), or both.
func trimBookmark(s string) string {
	if i := strings.Index(s, "@"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, "*")
}

// parseJJRemotes parses the output of `jj git remote list` into a name→url map.
// Each line is "<name> <url>"; malformed lines are skipped.
func parseJJRemotes(out string) map[string]string {
	remotes := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, url, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		remotes[name] = strings.TrimSpace(url)
	}
	return remotes
}
