package healthcheck

import (
	"encoding/json"
	"net/http"

	"github.com/bytearena/bytearena/common/utils"
)

type HealthCheckServer struct {
	Checkers []HealthCheckHandler
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

	for _, checker := range server.Checkers {
		err, checkerRes := checker()

		if err == nil {

			res.Checks = append(res.Checks, HealthChecks{
				Status: checkerRes,
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

func NewHealthCheckServer(port string) *HealthCheckServer {

	return &HealthCheckServer{
		port: port,
	}
}

func (server *HealthCheckServer) Listen() {
	http.HandleFunc("/health", server.httpHandler)

	err := http.ListenAndServe(":"+server.port, nil)
	utils.Check(err, "Failed to listen on :"+server.port)
}

func (server *HealthCheckServer) Register(name string, handler HealthCheckHandler) {
	server.Checkers = append(server.Checkers, handler)
}
