package arenamaster

import (
	"encoding/json"
	"log"

	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
)

type messageArenaHandshake struct {
	Uuid string `json:"uuid"`
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

func (server *Server) onMessage(msg mq.BrokerMessage, cb onLogic) {
	var message types.MQMessage
	err := json.Unmarshal(msg.Data, &message)
	if err != nil {
		log.Println(err)
		log.Println("ERROR:agent Invalid MQMessage " + string(msg.Data))
		return
	}

	onLogicResponseCallable := cb(server.state, message.Payload)

	if onLogicResponseCallable != nil {
		onLogicResponseCallable(server.brokerclient)
	}
}

func (server *Server) Start() ListeningChanStruct {
	log.Println("Listening")

	server.brokerclient.Subscribe("arena", "launch", func(msg mq.BrokerMessage) {
		server.onMessage(msg, onArenaLaunch)
	})

	server.brokerclient.Subscribe("arena", "handshake", func(msg mq.BrokerMessage) {
		server.onMessage(msg, onArenaHandshake)
	})

	server.listeningChan = make(ListeningChanStruct)

	return server.listeningChan
}

func (server *Server) Stop() {
	close(server.listeningChan)
}
