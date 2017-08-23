package arenamaster

import (
	"encoding/json"

	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

type ListeningChanStruct chan struct{}
type Server struct {
	listeningChan ListeningChanStruct
	brokerclient  *mq.Client
	graphqlclient *graphql.Client
	state         *State
}

func NewServer(mq *mq.Client, gql *graphql.Client) *Server {
	return &Server{
		brokerclient:  mq,
		graphqlclient: gql,
		state:         NewState(),
	}
}

func unmarshalMQMessage(msg mq.BrokerMessage) (error, *types.MQMessage) {
	var message types.MQMessage
	err := json.Unmarshal(msg.Data, &message)
	if err != nil {
		return err, nil
	} else {
		return nil, &message
	}
}

func (server *Server) Start() ListeningChanStruct {

	server.brokerclient.Subscribe("game", "launch", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
			return
		} else {
			onGameLaunch(server.state, message.Payload, server.brokerclient, server.graphqlclient)
		}
	})

	server.brokerclient.Subscribe("game", "launched", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
			return
		} else {
			onGameLaunched(server.state, message.Payload, server.brokerclient, server.graphqlclient)
		}
	})

	server.brokerclient.Subscribe("game", "handshake", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
			return
		} else {
			onGameHandshake(server.state, message.Payload)
		}
	})

	server.brokerclient.Subscribe("game", "stopped", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
			return
		} else {
			onGameStop(server.state, message.Payload, server.graphqlclient)
		}
	})

	server.listeningChan = make(ListeningChanStruct)

	utils.Debug("arenamaster", "Listening")

	return server.listeningChan
}

func (server *Server) Stop() {
	close(server.listeningChan)
}
