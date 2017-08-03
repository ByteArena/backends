package main

import (
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/healthcheck"
	"github.com/bytearena/bytearena/common/mq"

	"net/http"
)

func NewHealthCheck(brokerclient *mq.Client, graphqlclient graphql.Client, vizServerAddr string) *healthcheck.HealthCheckServer {
	healthCheckServer := healthcheck.NewHealthCheckServer()

	healthCheckServer.Register("mq", func() error {
		pingErr := brokerclient.Ping()

		if pingErr != nil {
			return pingErr
		} else {
			return nil
		}
	})

	healthCheckServer.Register("graphql", func() error {
		pingErr := graphqlclient.Ping()

		if pingErr != nil {
			return pingErr
		} else {
			return nil
		}
	})

	healthCheckServer.Register("viz-server", func() error {
		resp, err := http.Get(vizServerAddr)

		if err != nil && resp.StatusCode != 200 {
			return err
		} else {
			return nil
		}
	})

	return healthCheckServer
}
