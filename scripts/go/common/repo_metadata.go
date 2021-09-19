package common

import (
	"os/exec"
	"strings"
)

// MustGetGitCommitHash gets the Git commit hash of the current checkout.
func MustGetGitCommitHash() string {
	stdout, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		// We should always be run under a git checkout.
		panic(err)
	}

	return strings.TrimSpace(string(stdout))
}
