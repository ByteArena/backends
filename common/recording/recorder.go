package recording

import (
	"os"

	"github.com/bytearena/bytearena/common/utils"
)

type Recorder interface {
	Record(arenaId string, msg string) error
	Close()
}

func createFileIfNotExists(path string) {
	var _, err = os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		utils.Check(err, "Could not create file")

		defer file.Close()
	}

}
