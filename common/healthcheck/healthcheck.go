package healthcheck

import (
	"encoding/json"
	"net/http"

	"github.com/bytearena/bytearena/common/utils"
)

type HealthCheckServer struct {
	Checkers map[string]HealthCheckHandler
	port     string
}

type HealthChecks struct {
	Status bool
	Name   string
}

type HealthCheckHttpResponse struct {
	Checks     []HealthChecks
	StatusCode int
}

type HealthCheckHandler func() (err error, ok bool)

func (server *HealthCheckServer) httpHandler(w http.ResponseWriter, r *http.Request) {
	res := HealthCheckHttpResponse{
		Checks:     make([]HealthChecks, 0),
		StatusCode: 200,
	}

	for name, checker := range server.Checkers {
		err, checkerRes := checker()

		if err == nil {

			res.Checks = append(res.Checks, HealthChecks{
				Status: checkerRes,
				Name:   name,
			})
		} else {
			res.StatusCode = http.StatusInternalServerError
		}
	}

	data, err := json.Marshal(res)
	utils.Check(err, "Failed to marshal response")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.StatusCode)
	w.Write(data)
}

func NewHealthCheckServer() *HealthCheckServer {
	return &HealthCheckServer{
		port:     "8099",
		Checkers: make(map[string]HealthCheckHandler, 0),
	}
}

func (server *HealthCheckServer) Listen() {
	http.HandleFunc("/health", server.httpHandler)

	err := http.ListenAndServe(":"+server.port, nil)
	utils.Check(err, "Failed to listen on :"+server.port)
}

func (server *HealthCheckServer) Register(name string, handler HealthCheckHandler) {
	server.Checkers[name] = handler
}
