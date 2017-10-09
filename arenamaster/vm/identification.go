package vm

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	HEXCHARS = "0123456789abcdef"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func RandomHex(strlen int) string {
	result := make([]byte, strlen)
	for i := range result {
		result[i] = HEXCHARS[r.Intn(len(HEXCHARS))]
	}
	return string(result)
}

func GetRandomMAC() string {
	return fmt.Sprintf("00:f0:%s:%s:%s:%s", RandomHex(2), RandomHex(2), RandomHex(2), RandomHex(2))
}
