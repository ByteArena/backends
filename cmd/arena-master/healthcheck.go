package main

import (
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/healthcheck"
	"github.com/bytearena/bytearena/common/mq"
)

func NewHealthCheck(brokerclient *mq.Client, graphqlclient *graphql.Client) *healthcheck.HealthCheckServer {
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

	return healthCheckServer
}
