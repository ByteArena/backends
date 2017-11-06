package main

import (
	"os/exec"

	bettererrors "github.com/xtuc/better-errors"
)

func ensureDockerIsAvailable() {
	_, err := exec.LookPath("docker")

	if err != nil {
		failWith(
			bettererrors.NewFromString("Docker was not found in $PATH. Please install it."),
		)
	}
}
