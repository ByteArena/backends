package arenamaster

import (
	"github.com/bytearena/bytearena/common/mq"
)

type Res chan mq.BrokerMessage

type Listener struct {
	mqClient *mq.Client
}

func (l *Listener) subscribeToChannelAndGetChan(channel, topic string) Res {
	res := make(Res)

	l.mqClient.Subscribe(channel, topic, func(msg mq.BrokerMessage) {
		res <- msg
	})

	return res
}

func (l *Listener) ListenArenaAdd() Res {
	return l.subscribeToChannelAndGetChan("arena", "add")
}

func (l *Listener) ListenArenaHalt() Res {
	return l.subscribeToChannelAndGetChan("arena", "halt")
}
