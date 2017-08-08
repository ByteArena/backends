package arenamaster

import (
	"encoding/json"
	"log"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
)

type messageArenaHandshake struct {
	id string `json:"id"`
}

type ListeningChanStruct chan struct{}
type Server struct {
	listeningChan ListeningChanStruct
	brokerclient  *mq.Client
	state         *State
}

func NewServer(mq *mq.Client) *Server {
	return &Server{
		brokerclient: mq,
		state:        NewState(),
	}
}

func (server *Server) Start() ListeningChanStruct {
	log.Println("Listening")

	server.brokerclient.Subscribe("game", "launch", func(msg mq.BrokerMessage) {

		var message types.MQMessage
		err := json.Unmarshal(msg.Data, &message)
		if err != nil {
			log.Println(err)
			log.Println("ERROR:agent Invalid MQMessage " + string(msg.Data))
			return
		}

		onGameLaunch(server.state, message.Payload, server.brokerclient)
	})

	server.brokerclient.Subscribe("game", "handshake", func(msg mq.BrokerMessage) {

		var message types.MQMessage
		err := json.Unmarshal(msg.Data, &message)
		if err != nil {
			log.Println(err)
			log.Println("ERROR:agent Invalid MQMessage " + string(msg.Data))
			return
		}

		onGameHandshake(server.state, message.Payload)
	})

	server.brokerclient.Subscribe("game", "stoped", func(msg mq.BrokerMessage) {

		var message types.MQMessage
		err := json.Unmarshal(msg.Data, &message)
		if err != nil {
			log.Println(err)
			log.Println("ERROR:agent Invalid MQMessage " + string(msg.Data))
			return
		}

		onGameStop(server.state, message.Payload)
	})

	server.listeningChan = make(ListeningChanStruct)

	return server.listeningChan
}

func (server *Server) Stop() {
	close(server.listeningChan)
}
