package cmd

import (
	"os/exec"
	"strings"
)

func jjOutput(args ...string) string {
	out, err := exec.Command("jj", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
