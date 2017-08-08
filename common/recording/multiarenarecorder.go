package recording

import (
	"os"

	"github.com/bytearena/bytearena/common/types/mapcontainer"
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

func (r MutliArenaRecorder) Stop() {

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

func (r MutliArenaRecorder) RecordMetadata(UUID string, mapcontainer *mapcontainer.MapContainer) error {
	return nil
}

func (r MutliArenaRecorder) Close(UUID string) {
	handle, ok := r.fileHandles[UUID]

	// TODO(sven): bundle the zip here
	if ok {
		handle.Close()
	}
}

func (r MutliArenaRecorder) GetDirectory() string {
	return r.directory
}

// func (r SingleArenaRecorder) RecordMetadata(UUID string, mapcontainer *mapcontainer.MapContainer) error {
// 	filename := r.filename + ".meta"

// 	createFileIfNotExists(filename)

// 	metadata := RecordMetadata{
// 		MapContainer: mapcontainer,
// 		Date:         time.Now().Format(time.RFC3339),
// 	}

// 	data, err := json.Marshal(metadata)
// 	utils.Check(err, "could not marshall RecordMetadata")

// 	err = ioutil.WriteFile(filename, data, 0644)
// 	utils.Check(err, "could not write RecordMetadata file")

// 	utils.Debug("SingleArenaRecorder", "wrote record metadata for game "+UUID)

// 	r.recordMetadataData = string(data)

// 	return nil
// }
