package recording

import (
	"archive/zip"
	"os"

	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
)

type RecordMetadata struct {
	MapContainer *mapcontainer.MapContainer `json:"map"`
	Date         string                     `json:"date"`
}

type Recorder interface {
	RecordMetadata(UUID string, mapcontainer *mapcontainer.MapContainer) error
	Record(UUID string, msg string) error
	Close(UUID string)
	Stop()

	// Only used for MutliArenaRecorder
	GetDirectory() string
}

type ArchiveFile struct {
	Name string
	Body string
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

func MakeArchive(filename string, files []ArchiveFile) (error, *os.File) {
	createFileIfNotExists(filename)

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0600)
	utils.Check(err, "Could not open file")

	defer f.Close()

	w := zip.NewWriter(f)

	for _, file := range files {
		f, err := w.Create(file.Name)
		if err != nil {
			return err, nil
		}
		_, err = f.Write([]byte(file.Body))
		if err != nil {
			return err, nil
		}
	}

	err = w.Close()
	if err != nil {
		return err, nil
	}

	f.Sync()

	return nil, f
}
