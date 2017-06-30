package main

import (
	"github.com/bytearena/bytearena/common/healthcheck"
	"github.com/bytearena/bytearena/common/mq"
)

func StartHealthCheck(brokerclient *mq.Client) {
	healthCheckServer := healthcheck.NewHealthCheckServer()

	healthCheckServer.Register("mq", func() (err error, ok bool) {
		pingErr := brokerclient.Ping()

		if pingErr != nil {
			return pingErr, false
		} else {
			return nil, true
		}
	})

	healthCheckServer.Listen()
}
