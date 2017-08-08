package recording

import (
	"encoding/json"
	"time"

	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
)

type SingleArenaRecorder struct {
	buffer         string
	filename       string
	recordMetadata *RecordMetadata
}

func MakeSingleArenaRecorder(filename string) Recorder {

	return &SingleArenaRecorder{
		buffer:         "",
		filename:       filename,
		recordMetadata: nil,
	}
}

func (r *SingleArenaRecorder) Stop() {}

func (r *SingleArenaRecorder) Close(UUID string) {
	files := make([]ArchiveFile, 2)

	if r.recordMetadata == nil {
		panic("Missing RecordMetadata")
	}

	metadata, err := json.Marshal(*r.recordMetadata)
	utils.Check(err, "Coud not serialize RecordMetadata")

	files = append(files, ArchiveFile{
		Name: "RecordMetadata",
		Body: string(metadata),
	})

	files = append(files, ArchiveFile{
		Name: "Record",
		Body: r.buffer,
	})

	err, _ = MakeArchive(r.filename+".zip", files)
	utils.CheckWithFunc(err, func() string {
		return "could not create record archive" + err.Error()
	})

	utils.Debug("SingleArenaRecorder", "write record archive")
}

func (r *SingleArenaRecorder) RecordMetadata(UUID string, mapcontainer *mapcontainer.MapContainer) error {
	r.recordMetadata = &RecordMetadata{
		MapContainer: mapcontainer,
		Date:         time.Now().Format(time.RFC3339),
	}

	utils.Debug("SingleArenaRecorder", "created RecordMetadata")

	return nil
}

func (r *SingleArenaRecorder) Record(UUID string, msg string) error {
	r.buffer += msg + "\n"

	return nil
}

func (r *SingleArenaRecorder) GetDirectory() string {
	return ""
}
