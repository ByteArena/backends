package healthcheck

import (
	"encoding/json"
	"net/http"

	"github.com/bytearena/bytearena/common/utils"
)

type HealthCheckServer struct {
	Checkers map[string]HealthCheckHandler
	port     string
	listener *http.Server
}

type HealthChecks struct {
	Status bool
	Name   string
	Detail string
}

type HealthCheckHttpResponse struct {
	Checks     []HealthChecks
	StatusCode int
}

type HealthCheckHandler func() error

func (server *HealthCheckServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := HealthCheckHttpResponse{
		Checks:     make([]HealthChecks, 0),
		StatusCode: 200,
	}

	for name, checker := range server.Checkers {
		err := checker()

		if err == nil {

			res.Checks = append(res.Checks, HealthChecks{
				Status: true,
				Name:   name,
				Detail: "",
			})
		} else {
			res.StatusCode = http.StatusInternalServerError

			res.Checks = append(res.Checks, HealthChecks{
				Status: false,
				Name:   name,
				Detail: err.Error(),
			})
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

func (server *HealthCheckServer) Start() chan struct{} {

	server.listener = &http.Server{
		Addr:    ":" + server.port,
		Handler: server,
	}

	block := make(chan struct{})

	go func(block chan struct{}) {
		err := server.listener.ListenAndServe()
		utils.Check(err, "Failed to listen on :"+server.port)
		close(block)
	}(block)

	return block
}

func (server *HealthCheckServer) Stop() {
	server.listener.Shutdown(nil)
}

func (server *HealthCheckServer) Register(name string, handler HealthCheckHandler) {
	server.Checkers[name] = handler
}
