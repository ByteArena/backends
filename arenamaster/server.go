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

type ListeningChanStruct chan bool
type Server struct {
	stopChan       ListeningChanStruct
	brokerclient   *mq.Client
	graphqlclient  *graphql.Client
	state          *State
	influxdbClient *influxdb.Client
}

func NewServer(mq *mq.Client, gql *graphql.Client) *Server {
	stopChan := make(ListeningChanStruct)

	influxdbClient, influxdbClientErr := influxdb.NewClient("arenamaster")
	utils.Check(influxdbClientErr, "Unable to create influxdb client")

	s := &Server{
		brokerclient:   mq,
		graphqlclient:  gql,
		state:          NewState(),
		stopChan:       stopChan,
		influxdbClient: influxdbClient,
	}

	err := s.startStateReporting()

	utils.CheckWithFunc(err, func() string {
		panic("Could not start state reporting: " + err.Error())
	})

	return s
}

func (server *Server) startStateReporting() error {

	server.influxdbClient.Loop(func() {
		server.state.LockState()

		fields := map[string]interface{}{
			"state-idle":       len(server.state.idleArenas),
			"state-running":    len(server.state.runningArenas),
			"state-pending":    len(server.state.pendingArenas),
			"state-booting-vm": len(server.state.bootingVM),
		}

		server.state.UnlockState()

		server.influxdbClient.WriteAppMetric("arenamaster", fields)
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

func (server *Server) Run() {
	listener := MakeListener(server.brokerclient)

	for {
		select {
		case <-server.stopChan:
			return

		case <-listener.arenaAdd:
			inc++
			id := inc

			server.state.UpdateStateAddBootingVM(id)
			vm := vm.SpawnArena(id)
			server.state.UpdateStateVMBooted(id, vm)

		case msg := <-listener.arenaHalt:
			id, _ := strconv.Atoi((*msg.Payload)["id"].(string))

			if data, hasRunningVm := server.state.runningVM[id]; hasRunningVm {
				server.state.UpdateStateVMHalted(id)

				runningVM := data.(*vm.VM)
				runningVM.Quit()
			} else {
				utils.RecoverableError("vm", "Could not halt ("+strconv.Itoa(id)+"): VM is not running")
			}

		case msg := <-listener.gameLaunch:
			onGameLaunch(
				server.state,
				msg.Payload,
				server.brokerclient,
				server.graphqlclient,
			)

		case msg := <-listener.gameLaunched:
			onGameLaunched(
				server.state,
				msg.Payload,
				server.brokerclient,
				server.graphqlclient,
			)

		case msg := <-listener.gameHandshake:
			onGameHandshake(
				server.state,
				msg.Payload,
			)

		case msg := <-listener.gameHandshake:
			onGameStop(
				server.state,
				msg.Payload,
				server.graphqlclient,
			)
		}
	}
}

func (server *Server) Stop() {
	server.stopChan <- true
	server.influxdbClient.TearDown()

	close(server.stopChan)
}
