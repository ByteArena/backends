package utils

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

func FingerprintPublicKey(key string) (fingerprint string, comment string, err error) {

	keygenbin, err := exec.LookPath("ssh-keygen")
	if err != nil {
		return "", "", errors.New("Error: ssh-keygen not found in $PATH")
	}

	cmd := exec.Command(
		keygenbin,
		"-l",
		"-f", "-",
	)
	cmd.Stdin = bytes.NewBufferString(key)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", errors.New("Error: could not fingerprint publickey; " + err.Error() + "; " + string(output))
	}

	parts := strings.Split(string(output), " ")

	return parts[1], parts[2], nil
}
