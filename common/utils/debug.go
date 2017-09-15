package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Context map[string]interface{}

type Message struct {
	Time    string  `json:"time"`
	Service string  `json:"service"`
	Message string  `json:"message"`
	Context Context `json:"context"`
}

func Debug(service string, message string) {
	context := make(Context, 0)

	if hostname, err := os.Hostname(); err == nil {
		context["hostname"] = hostname
	}

	messageStruct := Message{
		Time:    time.Now().Format(time.RFC3339),
		Service: service,
		Message: message,
		Context: context,
	}

	data, _ := json.Marshal(messageStruct)

	fmt.Println(string(data))
}
