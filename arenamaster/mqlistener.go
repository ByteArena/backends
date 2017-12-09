package arenamaster

import (
	"encoding/json"

	bamq "github.com/bytearena/core/common/mq"
	"github.com/bytearena/core/common/types"
	"github.com/bytearena/core/common/utils"

	"github.com/bytearena/backends/common/mq"
)

type Res chan types.MQMessage

type Listener struct {
	arenaAdd           Res
	arenaHalt          Res
	gameLaunch         Res
	gameLaunched       Res
	gameHandshake      Res
	gameStopped        Res
	gameHealthcheckRes Res

	debugGetVMStatus Res
}

func MakeListener(mqClient *mq.Client) Listener {
	return Listener{
		arenaAdd:  subscribeToChannelAndGetChan(mqClient, "arena", "add"),
		arenaHalt: subscribeToChannelAndGetChan(mqClient, "arena", "halt"),

		gameLaunch:         subscribeToChannelAndGetChan(mqClient, "game", "launch"),
		gameLaunched:       subscribeToChannelAndGetChan(mqClient, "game", "launched"),
		gameHandshake:      subscribeToChannelAndGetChan(mqClient, "game", "handshake"),
		gameStopped:        subscribeToChannelAndGetChan(mqClient, "game", "stopped"),
		gameHealthcheckRes: subscribeToChannelAndGetChan(mqClient, "game", "healthcheck-res"),

		debugGetVMStatus: subscribeToChannelAndGetChan(mqClient, "debug", "getvmstatus"),
	}
}

func subscribeToChannelAndGetChan(mqClient *mq.Client, channel, topic string) Res {
	res := make(Res)

	err := mqClient.Subscribe(channel, topic, func(msg bamq.BrokerMessage) {
		var message types.MQMessage
		err := json.Unmarshal(msg.Data, &message)

		if err != nil {
			utils.RecoverableError("event listener", err.Error())
			return
		}

		res <- message
	})

	utils.Check(err, "Could not subscribe to mq")

	return res
}
