package utils

import (
	"path"
	"strconv"

	"github.com/bytearena/backends/dotgit/protocol"
)

func RepoRelPath(repo protocol.GitRepository) string {
	return path.Join(strconv.Itoa(int(repo.OwnerID)), strconv.Itoa(int(repo.ID))+".git")
}
