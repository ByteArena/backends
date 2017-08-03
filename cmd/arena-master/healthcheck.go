package main

import (
	"github.com/bytearena/bytearena/common/healthcheck"
	"github.com/bytearena/bytearena/common/mq"
)

func NewHealthCheck(brokerclient *mq.Client) *healthcheck.HealthCheckServer {
	healthCheckServer := healthcheck.NewHealthCheckServer()

	healthCheckServer.Register("mq", func() (err error, ok bool) {
		pingErr := brokerclient.Ping()

		if pingErr != nil {
			return pingErr, false
		} else {
			return nil, true
		}
	})

	return healthCheckServer
}
