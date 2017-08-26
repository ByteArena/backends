package recording

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"time"

	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
)

type RecordMetadata struct {
	MapContainer *mapcontainer.MapContainer `json:"map"`
	Date         string                     `json:"date"`
}

type RecorderInterface interface {
	RecordMetadata(UUID string, mapcontainer *mapcontainer.MapContainer) error
	Record(UUID string, msg string) error
	Close(UUID string)
	Stop()

	// Only used for MutliArenaRecorder
	GetDirectory() string
}

type ArchiveFile struct {
	Name string
	Fd   *os.File
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
	archiveFd, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	utils.Check(err, "Could not open file")

	defer archiveFd.Close()

	archiveWriter := zip.NewWriter(archiveFd)

	for _, file := range files {
		header := &zip.FileHeader{
			Name: file.Name,
			Method: zip.Deflate,
		}

		header.SetModTime(time.Now())

		writer, err := archiveWriter.CreateHeader(header)

		if err != nil {
			return err, nil
		}

		file.Fd.Seek(0, 0)
		_, err = io.Copy(writer, file.Fd)

		if err != nil {
			log.Println("copy failed")
			return err, nil
		}
	}

	err = archiveWriter.Close()
	if err != nil {
		return err, nil
	}

	archiveFd.Sync()

	return nil, archiveFd
}
