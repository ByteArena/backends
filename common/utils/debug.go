package utils

import (
	"encoding/json"
	"log"
	"os"
)

type Context map[string]interface{}

type Message struct {
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
		Service: service,
		Message: message,
		Context: context,
	}

	data, _ := json.Marshal(messageStruct)

	log.Println(string(data))
}
