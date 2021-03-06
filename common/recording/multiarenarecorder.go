package recording

import (
	"encoding/json"
	"os"
	"time"

	"github.com/bytearena/core/common/recording"
	"github.com/bytearena/core/common/types/mapcontainer"
	"github.com/bytearena/core/common/utils"
)

type MultiArenaRecorder struct {
	directory                 string
	recordFileHandles         map[string]*os.File
	recordMetadataFileHandles map[string]*os.File
}

func MakeMultiArenaRecorder(directory string) *MultiArenaRecorder {

	return &MultiArenaRecorder{
		recordFileHandles:         make(map[string]*os.File),
		recordMetadataFileHandles: make(map[string]*os.File),
		directory:                 directory,
	}
}

func (r *MultiArenaRecorder) Record(UUID string, msg string) error {
	handle, ok := r.recordFileHandles[UUID]

	if !ok {
		filename := r.directory + "/" + UUID + "-json"
		createFileIfNotExists(filename)

		var err error
		handle, err = os.OpenFile(filename, os.O_RDWR, 0600)
		utils.Check(err, "Could not open file")

		r.recordFileHandles[UUID] = handle
	}

	_, err := handle.WriteString(msg + "\n")
	utils.Check(err, "could write record entry")

	err = handle.Sync()

	utils.Check(err, "could not flush Record to disk")

	return err
}

func (r *MultiArenaRecorder) RecordMetadata(UUID string, mapcontainer *mapcontainer.MapContainer) error {
	_, ok := r.recordMetadataFileHandles[UUID]

	if !ok {

		filename := r.directory + "/" + UUID + "-json.meta"

		createFileIfNotExists(filename)

		file, err := os.OpenFile(filename, os.O_RDWR, 0644)
		utils.Check(err, "Could not open RecordMetadata file")

		metadata := recording.RecordMetadata{
			MapContainer: mapcontainer,
			Date:         time.Now().Format(time.RFC3339),
		}

		data, err := json.Marshal(metadata)
		utils.Check(err, "could not marshall RecordMetadata")

		_, err = file.Write(data)
		utils.Check(err, "could not write RecordMetadata file")

		err = file.Sync()
		utils.Check(err, "could not flush RecordMetadata to disk")

		utils.Debug("MutliArenaRecorder", "wrote record metadata for game "+UUID)

		r.recordMetadataFileHandles[UUID] = file
	}

	return nil
}

func (r *MultiArenaRecorder) GetFilePathForUUID(UUID string) string {
	return r.directory + "/" + UUID
}

func (r *MultiArenaRecorder) RecordExists(UUID string) bool {
	recordFile := r.GetFilePathForUUID(UUID)
	_, err := os.Stat(recordFile)

	return !os.IsNotExist(err)
}

func (r *MultiArenaRecorder) Close(UUID string) {
	recordHandle, okRecord := r.recordFileHandles[UUID]
	metadataHandle, okRecordMetadata := r.recordMetadataFileHandles[UUID]

	if okRecord && okRecordMetadata {
		files := make([]recording.ArchiveFile, 0)

		files = append(files, recording.ArchiveFile{
			Name: "RecordMetadata",
			Fd:   metadataHandle,
		})

		files = append(files, recording.ArchiveFile{
			Name: "Record",
			Fd:   recordHandle,
		})

		err, _ := recording.MakeArchive(r.directory+"/"+UUID, files)
		utils.CheckWithFunc(err, func() string {
			return "could not create record archive: " + err.Error()
		})

		recordHandle.Close()
		metadataHandle.Close()

		delete(r.recordFileHandles, UUID)
		delete(r.recordMetadataFileHandles, UUID)

		os.Remove(r.directory + "/" + UUID + "-json")
		os.Remove(r.directory + "/" + UUID + "-json.meta")

		utils.Debug("MutliArenaRecorder", "stopped recording for arena "+UUID)
	} else {
		utils.Debug("MutliArenaRecorder", "no running recording for arena "+UUID)
	}
}

func (r *MultiArenaRecorder) Stop() {

	for _, handle := range r.recordFileHandles {
		handle.Close()
	}

	for _, handle := range r.recordMetadataFileHandles {
		handle.Close()
	}
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
