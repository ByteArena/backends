package main

import (
	"net/http"
)

func homeHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, VIZ SERVER !"))
	}
}
