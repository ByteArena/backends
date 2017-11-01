package mappack

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

type MappackInMemoryArchive struct {
	Zip   zip.ReadCloser
	Files map[string]io.ReadCloser
}

func UnzipAndGetHandles(filename string) (*MappackInMemoryArchive, error) {
	mappackInMemoryArchive := &MappackInMemoryArchive{
		Files: make(map[string]io.ReadCloser),
	}

	reader, err := zip.OpenReader(filename)

	if err != nil {
		return nil, errors.Wrapf(err, "Could not open archive (%s)", filename)
	}

	for _, file := range reader.File {
		fd, err := file.Open()

		if err != nil {
			return nil, errors.Wrapf(err, "Could not open file in archive (%s)", file.Name)
		}

		mappackInMemoryArchive.Files[file.Name] = fd
	}

	return mappackInMemoryArchive, nil
}

func (m *MappackInMemoryArchive) Open(name string) (*bufio.Reader, error) {
	if file, hasFile := m.Files[name]; hasFile {
		return bufio.NewReader(file), nil
	}

	return nil, errors.New(fmt.Sprintf("File %s not found", name))
}

func (m *MappackInMemoryArchive) Close() {
	for _, fd := range m.Files {
		fd.Close()
	}

	m.Zip.Close()
}
