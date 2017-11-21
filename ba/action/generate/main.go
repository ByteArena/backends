package generate

import (
	"fmt"
	"os/exec"
	"path"

	"github.com/bytearena/bytearena/ba/action/build"
	bettererrors "github.com/xtuc/better-errors"
)

var (
	samples = map[string]string{
		"nodejs": "https://github.com/xtuc/sample-nodejs-agent.git",
	}
)

func cloneRepo(dest, url string) (string, error) {
	cmd := exec.Command("git", "clone", url, dest)

	stdout, stderr := cmd.CombinedOutput()

	err := cmd.Run()

	if err != nil && stderr != nil {
		return string(stdout), stderr
	}

	cmd = exec.Command("rm", "-rf", path.Join(dest, "./.git"))

	stdout, stderr = cmd.CombinedOutput()

	err = cmd.Run()

	if err != nil && stderr != nil {
		return string(stdout), stderr
	}

	return "", nil
}

func Main(name string) error {

	if name == "" {
		name = "unknown"
	}

	dest := name + "-agent"

	if url, hasSample := samples[name]; hasSample {
		out, err := cloneRepo(dest, url)

		if err != nil {
			return bettererrors.
				NewFromErr(err).
				SetContext("error", out)
		}
	} else {
		berror := bettererrors.
			New("Unknown sample").
			SetContext("name", name)

		return berror
	}

	fmt.Println(dest, "has been created")

	// Build agent
	err := build.Main(dest)

	if err != nil {
		berror := bettererrors.
			New("ba build failed").
			With(err)

		return berror
	}

	return nil
}
