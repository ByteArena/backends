package main

import (
	"github.com/bytearena/bytearena/common/healthcheck"
	"github.com/bytearena/bytearena/common/mq"
)

func NewHealthCheck(brokerclient *mq.Client) *healthcheck.HealthCheckServer {
	healthCheckServer := healthcheck.NewHealthCheckServer()

	healthCheckServer.Register("mq", func() error {
		pingErr := brokerclient.Ping()

		if pingErr != nil {
			return pingErr
		} else {
			return nil
		}
	})

	return healthCheckServer
}
