package main

import (
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"

	"errors"
	"os/exec"
)

func StartMQHealthCheckServer(brokerclient *mq.Client, graphqlclient graphql.Client, id string) {

	testMq := func() error {
		pingErr := brokerclient.Ping()

		if pingErr != nil {
			return pingErr
		} else {
			return nil
		}
	}

	testGraphql := func() error {
		pingErr := graphqlclient.Ping()

		if pingErr != nil {
			return pingErr
		} else {
			return nil
		}
	}

	testDocker := func() error {
		dockerBin, LookPatherr := exec.LookPath("docker")

		if LookPatherr != nil {
			return LookPatherr
		}

		command := exec.Command(dockerBin, "ps")

		out, stderr := command.CombinedOutput()

		if stderr != nil {
			return errors.New(string(out))
		} else {
			return nil
		}
	}

	brokerclient.Subscribe("game", "healthcheck", func(msg mq.BrokerMessage) {
		var status = "OK"

		utils.Debug("healthcheck", "Probing")

		if err := testMq(); err != nil {
			status = "NOK"
		}

		if err := testGraphql(); err != nil {
			status = "NOK"
		}

		if err := testDocker(); err != nil {
			status = "NOK"
		}

		handshakeErr := brokerclient.Publish("game", "healthcheck-res", types.NewMQMessage(
			"arena-server",
			"healthcheck",
		).SetPayload(types.MQPayload{
			"health": status,
			"id":     id,
		}))

		utils.Check(handshakeErr, "Could not send health")
	})
}
