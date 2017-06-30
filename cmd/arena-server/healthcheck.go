package main

import (
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/healthcheck"
	"github.com/bytearena/bytearena/common/mq"
)

func StartHealthCheck(brokerclient *mq.Client, graphqlclient graphql.Client) {
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

	healthCheckServer.Listen()
}
