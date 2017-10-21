package utils

import "bufio"

func ReadFullLine(r *bufio.Reader) (string, error) {
	line, isPrefix, readErr := r.ReadLine()

	if readErr != nil {
		return "", readErr
	}

	if len(line) == 0 {
		return "", nil
	}

	var buf string

	if !isPrefix {
		buf = string(line)
	} else {
		buf = string(line)

		for isPrefix && readErr == nil {
			line, isPrefix, readErr = r.ReadLine()
			buf += string(line)
		}
	}

	return buf, nil
}
