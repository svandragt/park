package cmd

// currentRemote returns the origin URL, or "" if there is none.
func currentRemote() string {
	return gitOutput("remote", "get-url", "origin")
}

// currentBranch returns the current branch, or "" if HEAD is detached.
func currentBranch() string {
	return gitOutput("branch", "--show-current")
}
