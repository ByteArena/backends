package utils

import (
	"path"

	"github.com/kardianos/osext"
)

func GetAbsoluteDir(relative string) string {

	exfolder, err := osext.ExecutableFolder()
	Check(err, "Cannot get absolute dir for "+relative)

	return path.Join(exfolder, relative)
}
