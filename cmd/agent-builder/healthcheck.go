package main

import (
	"net/http"

	"github.com/bytearena/bytearena/common/healthcheck"
)

func PingRegistry(host string) (error, bool) {
	req, err := http.Get("http://" + host + "/v2/")

	if err != nil && req.StatusCode != 200 {
		return err, false
	}

	return nil, true
}

func StartHealthCheck(registryHost string) {
	healthCheckServer := healthcheck.NewHealthCheckServer()

	// FIXME(sven): doesn't work be cause the registryHost passed in the env
	// is used by the docker client which runs on the host.
	// We pass localhost and it's only accessible from the host

	// healthCheckServer.Register("Docker registry", func() (err error, ok bool) {
	// 	pingErr, status := PingRegistry(registryHost)

	// 	if pingErr != nil {
	// 		return pingErr, status
	// 	} else {
	// 		return nil, status
	// 	}
	// })

	healthCheckServer.Listen()
}
