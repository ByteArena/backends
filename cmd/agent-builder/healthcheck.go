package main

import (
	"errors"
	"net/http"
	"os/exec"

	"github.com/bytearena/bytearena/common/healthcheck"
	"github.com/bytearena/bytearena/common/mq"
)

func PingRegistry(host string) (error, bool) {
	req, err := http.Get("http://" + host + "/v2/")

	if err != nil && req.StatusCode != 200 {
		return err, false
	}

	return nil, true
}

func NewHealthCheck(brokerclient *mq.Client, registryHost string) *healthcheck.HealthCheckServer {
	healthCheckServer := healthcheck.NewHealthCheckServer()

	healthCheckServer.Register("mq", func() (err error, ok bool) {
		pingErr := brokerclient.Ping()

		if pingErr != nil {
			return pingErr, false
		} else {
			return nil, true
		}
	})

	healthCheckServer.Register("docker", func() (err error, ok bool) {
		dockerBin, LookPatherr := exec.LookPath("docker")

		if LookPatherr != nil {
			return LookPatherr, false
		}

		command := exec.Command(dockerBin, "ps")

		out, stderr := command.CombinedOutput()

		if stderr != nil {
			return errors.New(string(out)), false
		} else {
			return nil, true
		}
	})

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

	return healthCheckServer
}
