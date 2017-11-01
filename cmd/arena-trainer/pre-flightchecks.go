package main

import (
	"os/exec"

	"github.com/pkg/errors"
)

func ensureDockerIsAvailable() {
	_, err := exec.LookPath("docker")

	if err != nil {
		failWith(errors.New("Docker was not found in $PATH. Please install it."))
	}
}
