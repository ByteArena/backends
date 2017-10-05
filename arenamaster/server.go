package arenamaster

import (
	"encoding/json"
	"strconv"

	"github.com/bytearena/bytearena/arenamaster/vm"
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/influxdb"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
)

var inc = 0

type ListeningChanStruct chan struct{}
type Server struct {
	listeningChan ListeningChanStruct
	brokerclient  *mq.Client
	graphqlclient *graphql.Client
	state         *State
}

func NewServer(mq *mq.Client, gql *graphql.Client) *Server {
	s := &Server{
		brokerclient:  mq,
		graphqlclient: gql,
		state:         NewState(),
	}

	influxdbClient, influxdbClientErr := influxdb.NewClient("arenamaster")
	utils.Check(influxdbClientErr, "Unable to create influxdb client")

	err := s.startStateReporting(influxdbClient)

	utils.CheckWithFunc(err, func() string {
		panic("Could not start state reporting: " + err.Error())
	})

	return s
}

func (server *Server) startStateReporting(influxdbClient *influxdb.Client) error {

	influxdbClient.Loop(func() {
		server.state.LockState()

		fields := map[string]interface{}{
			"state-idle":       len(server.state.idleArenas),
			"state-running":    len(server.state.runningArenas),
			"state-pending":    len(server.state.pendingArenas),
			"state-booting-vm": len(server.state.bootingVM),
		}

		server.state.UnlockState()

		influxdbClient.WriteAppMetric("arenamaster", fields)
	})

	return nil
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
	listener := MakeListener(server.brokerclient)

	for {
		select {
		case <-listener.arenaAdd:
			inc++
			id := inc

			server.state.UpdateAddBootingVM(id)
			vm := vm.SpawnArena(id)
			server.state.UpdateVMBooted(id, vm)

		case msg := <-listener.arenaHalt:
			id, _ := strconv.Atoi((*msg.Payload)["id"].(string))

			if data, hasRunningVm := server.state.runningVM[id]; hasRunningVm {
				server.state.UpdateVMHalted(id)

				runningVM := data.(*vm.VM)
				runningVM.Quit()
			} else {
				utils.RecoverableError("vm", "Could not halt ("+strconv.Itoa(id)+"): VM is not running")
			}
		}
	}

	server.brokerclient.Subscribe("game", "launch", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
		} else {
			onGameLaunch(server.state, message.Payload, server.brokerclient, server.graphqlclient)
		}
	})

	server.brokerclient.Subscribe("game", "launched", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
		} else {
			onGameLaunched(server.state, message.Payload, server.brokerclient, server.graphqlclient)
		}
	})

	server.brokerclient.Subscribe("game", "handshake", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
		} else {
			onGameHandshake(server.state, message.Payload)
		}
	})

	server.brokerclient.Subscribe("game", "stopped", func(msg mq.BrokerMessage) {
		err, message := unmarshalMQMessage(msg)

		if err != nil {
			utils.Debug("arenamaster", "Invalid MQMessage "+string(msg.Data))
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
