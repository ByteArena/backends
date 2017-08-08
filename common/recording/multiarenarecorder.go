package recording

import (
	"os"

	"github.com/bytearena/bytearena/common/utils"
)

type MutliArenaRecorder struct {
	directory   string
	fileHandles map[string]*os.File
}

func MakeMultiArenaRecorder(directory string) Recorder {

	return MutliArenaRecorder{
		fileHandles: make(map[string]*os.File),
		directory:   directory,
	}
}

func (r MutliArenaRecorder) Close() {

	for _, handle := range r.fileHandles {
		handle.Close()
	}
}

func createFileHandle(filename string) *os.File {
	createFileIfNotExists(filename)

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0600)
	utils.Check(err, "Could not open file")

	return f
}

func (r MutliArenaRecorder) Record(UUID string, msg string) error {
	handle, ok := r.fileHandles[UUID]

	if !ok {
		handle = createFileHandle(r.directory + "/record-" + UUID + ".bin")
		r.fileHandles[UUID] = handle
	}

	_, err := handle.WriteString(msg + "\n")

	return err
}
