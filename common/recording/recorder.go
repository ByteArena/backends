package recording

import (
	"os"

	"github.com/bytearena/bytearena/common/utils"
)

type Recorder interface {
	Record(UUID string, msg string) error
	Close()

	// Only used for MutliArenaRecorder
	GetDirectory() string
}

func createFileIfNotExists(path string) {
	var _, err = os.Stat(path)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(path)
		utils.Check(err, "Could not create file")

		utils.Debug("recorder", "created record file "+path)

		defer file.Close()
	}

}