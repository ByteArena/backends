package recording

import (
	"os"

	"github.com/bytearena/bytearena/common/utils"
)

type SingleArenaRecorder struct {
	fileHandle *os.File
}

func MakeSingleArenaRecorder(filename string) Recorder {
	createFileIfNotExists(filename)

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0600)
	utils.Check(err, "Could not open file")

	return SingleArenaRecorder{
		fileHandle: f,
	}
}

func (r SingleArenaRecorder) Close() {
	r.fileHandle.Close()
}

func (r SingleArenaRecorder) Record(UUID string, msg string) error {
	_, err := r.fileHandle.WriteString(msg + "\n")

	return err
}

func (r SingleArenaRecorder) GetDirectory() string {
	return ""
}
