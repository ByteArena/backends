package replay

import (
	"bufio"
	"io"
	"os"

	"github.com/bytearena/bytearena/common/utils"
)

type OnMessageFunc func(string, bool, string)

func Read(filename string, debug bool, UUID string, onMessage OnMessageFunc) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0755)

	utils.CheckWithFunc(err, func() string {
		return "File open failed: " + err.Error()
	})

	reader := bufio.NewReader(file)

	for {
		line, isPrefix, readErr := reader.ReadLine()

		if len(line) == 0 {
			continue
		}

		if readErr == io.EOF {
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
