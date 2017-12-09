package main

import (
	"errors"
	"os/exec"
	"time"

	"github.com/bytearena/backends/common/graphql"
	"github.com/bytearena/backends/common/mq"

	coremq "github.com/bytearena/core/common/mq"
	"github.com/bytearena/core/common/types"
	"github.com/bytearena/core/common/utils"
)

var (
	startedAt = time.Now()
)

func StartMQHealthCheckServer(brokerclient *mq.Client, graphqlclient graphql.Client, id string, duration time.Duration) {

	testTimeElapsed := func() error {
		now := time.Now()

		if now.Sub(startedAt) >= duration {
			return errors.New("Game is over")
		} else {
			return nil
		}
	}

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

	brokerclient.Subscribe("game", "healthcheck", func(msg coremq.BrokerMessage) {
		var status = "OK"

		if err := testTimeElapsed(); err != nil {
			status = "NOK"
		}

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
