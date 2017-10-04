package arenamaster

import (
	"github.com/bytearena/bytearena/common/mq"
)

type Res chan mq.BrokerMessage

type Listener struct {
	arenaAdd  Res
	arenaHalt Res
}

func MakeListener(mqClient *mq.Client) Listener {
	return Listener{
		arenaAdd:  subscribeToChannelAndGetChan(mqClient, "arena", "add"),
		arenaHalt: subscribeToChannelAndGetChan(mqClient, "arena", "halt"),
	}
}

func subscribeToChannelAndGetChan(mqClient *mq.Client, channel, topic string) Res {
	res := make(Res)

	mqClient.Subscribe(channel, topic, func(msg mq.BrokerMessage) {
		res <- msg
	})

	return res
}
