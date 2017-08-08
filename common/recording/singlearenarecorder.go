package recording

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
)

type SingleArenaRecorder struct {
	filename   string
	recordFile *os.File
}

func MakeSingleArenaRecorder(filename string) Recorder {
	f, err := os.OpenFile(os.TempDir()+"/"+filename, os.O_RDWR|os.O_CREATE, 0600)
	utils.Check(err, "Could not open file")

	return &SingleArenaRecorder{
		filename:   filename,
		recordFile: f,
	}
}

func (r *SingleArenaRecorder) Stop() {

	err := os.Remove(os.TempDir() + "/" + r.filename + ".meta")
	if err != nil {
		log.Println("Could not remove record temporary meta file: " + err.Error())
	}

	err = os.Remove(os.TempDir() + "/" + r.filename)
	if err != nil {
		log.Println("Could not remove record temporary file: " + err.Error())
	}
}

func (r *SingleArenaRecorder) Close(UUID string) {

	files := make([]ArchiveFile, 2)

	// files = append(files, ArchiveFile{
	// 	Name: "RecordMetadata",
	// 	Fd: r.MapContainer,
	// })

	files = append(files, ArchiveFile{
		Name: "Record",
		Fd:   r.recordFile,
	})

	err, _ := MakeArchive(r.filename+".zip", files)
	utils.CheckWithFunc(err, func() string {
		return "could not create record archive: " + err.Error()
	})

	r.recordFile.Close()

	utils.Debug("SingleArenaRecorder", "write record archive")
}

func (r SingleArenaRecorder) RecordMetadata(UUID string, mapcontainer *mapcontainer.MapContainer) error {
	filename := os.TempDir() + "/" + r.filename + ".meta"

	createFileIfNotExists(filename)

	metadata := RecordMetadata{
		MapContainer: mapcontainer,
		Date:         time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(metadata)
	utils.Check(err, "could not marshall RecordMetadata")

	err = ioutil.WriteFile(filename, data, 0644)
	utils.Check(err, "could not write RecordMetadata file")

	utils.Debug("SingleArenaRecorder", "wrote record metadata for game "+UUID)

	return nil
}

func (r *SingleArenaRecorder) Record(UUID string, msg string) error {
	_, err := r.recordFile.WriteString(msg + "\n")

	return err
}

func (r *SingleArenaRecorder) GetDirectory() string {
	return ""
}
