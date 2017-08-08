package recording

import (
	"encoding/json"
	"os"
	"time"

	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
)

type SingleArenaRecorder struct {
	filename           string
	recordFile         *os.File
	recordMetadataFile *os.File
}

func MakeSingleArenaRecorder(filename string) Recorder {
	f, err := os.Create("Record")
	utils.Check(err, "Could not open file")

	return &SingleArenaRecorder{
		filename:   filename,
		recordFile: f,
	}
}

func (r *SingleArenaRecorder) Stop() {
}

func (r *SingleArenaRecorder) Close(UUID string) {
	files := make([]ArchiveFile, 0)

	files = append(files, ArchiveFile{
		Name: "RecordMetadata",
		Fd:   r.recordMetadataFile,
	})

	files = append(files, ArchiveFile{
		Name: "Record",
		Fd:   r.recordFile,
	})

	err, _ := MakeArchive(r.filename, files)
	utils.CheckWithFunc(err, func() string {
		return "could not create record archive: " + err.Error()
	})

	r.recordFile.Close()
	r.recordMetadataFile.Close()

	utils.Debug("SingleArenaRecorder", "write record archive")
}

func (r *SingleArenaRecorder) RecordMetadata(UUID string, mapcontainer *mapcontainer.MapContainer) error {
	file, err := os.Create("RecordMetadata")
	utils.Check(err, "Could not open RecordMetadata temporary file")

	metadata := RecordMetadata{
		MapContainer: mapcontainer,
		Date:         time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(metadata)
	utils.Check(err, "could not marshall RecordMetadata")

	_, err = file.Write(data)

	utils.Debug("SingleArenaRecorder", "wrote record metadata for game "+UUID)

	r.recordMetadataFile = file

	return nil
}

func (r *SingleArenaRecorder) Record(UUID string, msg string) error {
	_, err := r.recordFile.WriteString(msg + "\n")

	return err
}

func (r *SingleArenaRecorder) GetDirectory() string {
	return ""
}
