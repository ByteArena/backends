package train

import (
	"os/exec"

	"github.com/bytearena/bytearena/common/utils"
	bettererrors "github.com/xtuc/better-errors"
)

func runPreflightChecks() {
	ensureDockerIsAvailable()
}

func ensureDockerIsAvailable() {
	_, err := exec.LookPath("docker")

	if err != nil {
		utils.FailWith(
			bettererrors.NewFromString("Docker was not found in $PATH. Please install it."),
		)
	}
}
