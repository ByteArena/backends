package main

import (
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/healthcheck"
	"github.com/bytearena/bytearena/common/mq"

	"errors"
	"os/exec"
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

	healthCheckServer.Start()
}
