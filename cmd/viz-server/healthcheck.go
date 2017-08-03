package main

import (
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/healthcheck"
	"github.com/bytearena/bytearena/common/mq"

	"net/http"
)

func NewHealthCheck(brokerclient *mq.Client, graphqlclient graphql.Client, vizServerAddr string) *healthcheck.HealthCheckServer {
	healthCheckServer := healthcheck.NewHealthCheckServer()

	healthCheckServer.Register("mq", func() (err error, ok bool) {
		pingErr := brokerclient.Ping()

		if pingErr != nil {
			return pingErr, false
		} else {
			return nil, true
		}
	})

	healthCheckServer.Register("graphql", func() (err error, ok bool) {
		pingErr, status := graphqlclient.Ping()

		if pingErr != nil {
			return pingErr, status
		} else {
			return nil, status
		}
	})

	healthCheckServer.Register("viz-server", func() (err error, ok bool) {
		resp, err := http.Get(vizServerAddr)

		if err != nil && resp.StatusCode != 200 {
			return err, false
		} else {
			return nil, true
		}
	})

	return healthCheckServer
}
