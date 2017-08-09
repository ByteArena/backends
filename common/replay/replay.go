package replay

import (
	"archive/zip"
	"bufio"
	"errors"
	"io"
	"io/ioutil"

	"github.com/bytearena/bytearena/common/utils"
)

type OnEventFunc func(string, bool, string)

type rawRecordHandles struct {
	recordMetadata io.ReadCloser
	record         io.ReadCloser
	zip            *zip.ReadCloser
}

func Read(filename string, debug bool, UUID string, onMessage OnEventFunc, onMap OnEventFunc) {
	err, rawRecordHandles := unzip(filename)
	utils.Check(err, "Could not decode archive")

	go func() {
		reader := bufio.NewReader(rawRecordHandles.recordMetadata)
		metadata, err := ioutil.ReadAll(reader)

		utils.Check(err, "Could not read metadata")

		onMap(string(metadata), debug, UUID)

		defer rawRecordHandles.recordMetadata.Close()
	}()

	reader := bufio.NewReader(rawRecordHandles.record)

	for {
		line, isPrefix, readErr := reader.ReadLine()

		if len(line) == 0 {
			continue
		}

		if readErr == io.EOF {
			rawRecordHandles.zip.Close()
			rawRecordHandles.record.Close()
			return
		}

		if !isPrefix {
			onMessage(string(line), debug, UUID)
		} else {
			buf := append([]byte(nil), line...)
			for isPrefix && err == nil {
				line, isPrefix, err = reader.ReadLine()
				buf = append(buf, line...)
			}

			onMessage(string(buf), debug, UUID)
		}
	}
}

func unzip(filename string) (error, *rawRecordHandles) {
	rawRecordHandles := &rawRecordHandles{}

	reader, err := zip.OpenReader(filename)

	if err != nil {
		return errors.New("could not open zip file (" + err.Error() + ")"), nil
	}

	rawRecordHandles.zip = reader

	for _, file := range reader.File {
		fd, err := file.Open()

		if err != nil {
			return err, nil
		}

		if file.Name == "Record" {
			rawRecordHandles.record = fd
		} else if file.Name == "RecordMetadata" {
			rawRecordHandles.recordMetadata = fd
		}
	}

	return nil, rawRecordHandles
}
