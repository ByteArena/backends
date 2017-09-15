package replay

import (
	"archive/zip"
	"bufio"
	"errors"
	"io"
	"io/ioutil"

	"github.com/bytearena/bytearena/common/utils"
)

type rawRecordHandles struct {
	recordMetadata io.ReadCloser
	record         io.ReadCloser
	zip            *zip.ReadCloser
}

type ReplayMessage struct {
	Line string
	UUID string
}

type Replayer struct {
	stopChannel      chan bool
	debug            bool
	UUID             string
	filename         string
	streamingChannel chan *ReplayMessage
	rawRecordHandles rawRecordHandles
}

func NewReplayer(filename string, debug bool, UUID string) *Replayer {

	err, rawRecordHandles := unzip(filename)
	utils.Check(err, "Could not decode archive")

	return &Replayer{
		streamingChannel: make(chan *ReplayMessage),
		debug:            debug,
		UUID:             UUID,
		filename:         filename,
		rawRecordHandles: *rawRecordHandles,
	}
}

func (r *Replayer) ReadMap() chan string {
	mapChannel := make(chan string)

	go func() {

		reader := bufio.NewReader(r.rawRecordHandles.recordMetadata)
		metadata, err := ioutil.ReadAll(reader)

		utils.Check(err, "Could not read metadata")

		mapChannel <- string(metadata)

		defer r.rawRecordHandles.recordMetadata.Close()
	}()

	return mapChannel
}

func (r *Replayer) Read() chan *ReplayMessage {
	reader := bufio.NewReader(r.rawRecordHandles.record)

	go func() {
		for {
			line, isPrefix, readErr := reader.ReadLine()

			if len(line) == 0 {
				continue
			}

			if readErr == io.EOF {
				r.rawRecordHandles.zip.Close()
				r.rawRecordHandles.record.Close()
				r.streamingChannel <- nil
			}

			if !isPrefix {
				r.streamingChannel <- &ReplayMessage{
					Line: string(line),
					UUID: r.UUID,
				}

			} else {
				buf := append([]byte(nil), line...)

				for isPrefix && readErr == nil {
					line, isPrefix, readErr = reader.ReadLine()
					buf = append(buf, line...)
				}

				r.streamingChannel <- &ReplayMessage{
					Line: string(buf),
					UUID: r.UUID,
				}

			}
		}
	}()

	return r.streamingChannel
}

func (r *Replayer) Stop() {
	utils.Debug("recorder", "stop replayer")
	r.stopChannel <- true
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
