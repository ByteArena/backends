package arenamaster

import (
	"encoding/json"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

type Res chan types.MQMessage

type Listener struct {
	arenaAdd      Res
	arenaHalt     Res
	gameLaunch    Res
	gameLaunched  Res
	gameHandshake Res
	gameStopped   Res
}

func MakeListener(mqClient *mq.Client) Listener {
	return Listener{
		arenaAdd:  subscribeToChannelAndGetChan(mqClient, "arena", "add"),
		arenaHalt: subscribeToChannelAndGetChan(mqClient, "arena", "halt"),

		gameLaunch:    subscribeToChannelAndGetChan(mqClient, "game", "launch"),
		gameLaunched:  subscribeToChannelAndGetChan(mqClient, "game", "launched"),
		gameHandshake: subscribeToChannelAndGetChan(mqClient, "game", "handshake"),
		gameStopped:   subscribeToChannelAndGetChan(mqClient, "game", "stopped"),
	}
}

func subscribeToChannelAndGetChan(mqClient *mq.Client, channel, topic string) Res {
	res := make(Res)

	err := mqClient.Subscribe(channel, topic, func(msg mq.BrokerMessage) {
		var message types.MQMessage
		err := json.Unmarshal(msg.Data, &message)

		if err != nil {
			utils.RecoverableError("event listener", err.Error())
		}

		res <- message
	})

	utils.Check(err, "Could not subscribe to mq")

	return res
}
