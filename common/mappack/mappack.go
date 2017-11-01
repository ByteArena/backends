package mappack

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

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

func (m *MappackInMemoryArchive) Open(name string) ([]byte, error) {
	if file, hasFile := m.Files[name]; hasFile {
		return ioutil.ReadAll(file)
	}

	return nil, errors.New(fmt.Sprintf("File %s not found", name))
}

func (m *MappackInMemoryArchive) Close() {
	for _, fd := range m.Files {
		fd.Close()
	}

	m.Zip.Close()
}

func (m *MappackInMemoryArchive) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "model.json") {
		r.URL.Path += ".gz"
		w.Header().Set("Content-Encoding", "gzip")
	}

	content, err := m.Open(r.URL.Path)

	if err != nil {
		w.Write([]byte(err.Error()))
	} else {
		ctype := mime.TypeByExtension(filepath.Ext(r.URL.Path))

		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Content-Size", strconv.Itoa(len(content)))
		w.Write(content)
	}
}
