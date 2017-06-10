package utils

import (
	"errors"
	"os/exec"
	"path"

	"github.com/bytearena/bytearena/dotgit/config"
	"github.com/bytearena/bytearena/dotgit/protocol"
)

func InitBareGitRepository(repo protocol.GitRepository) error {

	cnf := config.GetConfig()
	repoAbsPath := path.Join(cnf.GetGitRepositoriesPath(), RepoRelPath(repo))

	gitbin, err := exec.LookPath("git")
	if err != nil {
		return errors.New("Error: git not found in $PATH")
	}

	// TODO: use git template here
	cmd := exec.Command(
		gitbin,
		"init", "--bare", "--template", "/home/git/template",
		repoAbsPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return errors.New("Error: git init --bare failed; " + err.Error() + "; " + string(output))
	}

	return nil
}
