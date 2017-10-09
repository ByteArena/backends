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

		fields := make(map[string]interface{})

		// Transform map[string]int into map[string]interface{}
		// it works somehow
		for k, v := range server.state.DebugGetStateDistribution() {
			fields[k] = v
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
			vm, err := vm.SpawnArena(id)

			if err != nil {
				utils.RecoverableError("vm", "Could not start ("+strconv.Itoa(id)+"): "+err.Error())
				server.state.UpdateStateVMErrored(id)
			} else {
				go func() {
					err := vm.WaitUntilBooted()

					if err != nil {
						utils.RecoverableError("vm", "Could not wait until VM is booted")
						server.state.UpdateStateVMErrored(id)
					}

					server.state.UpdateStateVMBooted(id, vm)
				}()
			}

		case msg := <-listener.arenaHalt:
			id, _ := strconv.Atoi((*msg.Payload)["id"].(string))

			if data := server.state.QueryState(id, STATE_RUNNING_VM); data != nil {
				server.state.UpdateStateVMHalted(id)

				runningVM := data.(*vm.VM)
				runningVM.Quit()
			} else {
				utils.RecoverableError("vm", "Could not halt ("+strconv.Itoa(id)+"): VM is not running")
			}

		case msg := <-listener.gameLaunch:
			gameid, _ := (*msg.Payload)["id"].(string)
			element := server.state.FindState(STATE_IDLE_ARENA)

			if element != nil {
				vm := element.(*vm.VM)
				server.state.UpdateStateTriedLaunchArena(vm.Config.Id)

				onGameLaunch(
					gameid,
					server.brokerclient,
					server.graphqlclient,
					vm,
				)

			} else {
				utils.RecoverableError("vm", "Could not launch game: no arena is currently idle")
			}

		case msg := <-listener.gameLaunched:
			mac, _ := (*msg.Payload)["arenaserveruuid"].(string)
			gameid, _ := (*msg.Payload)["id"].(string)
			vm := FindVMByMAC(server.state, mac)

			if vm != nil {
				server.state.UpdateStateConfirmedLaunchArena(vm.Config.Id)
				go onGameLaunched(gameid, mac, server.graphqlclient)

				utils.Debug("master", mac+" launched")

			} else {
				utils.RecoverableError("game-launched", "VM with MAC ("+mac+") does not exists")
			}

		case msg := <-listener.gameHandshake:
			mac, _ := (*msg.Payload)["arenaserveruuid"].(string)
			vm := FindVMByMAC(server.state, mac)

			if vm != nil {
				server.state.UpdateStateAddIdleArena(vm.Config.Id)
				utils.Debug("master", mac+" joined")
			} else {
				utils.RecoverableError("game-handshake", "VM with MAC ("+mac+") does not exists")
			}

		case msg := <-listener.gameStopped:
			gameid, _ := (*msg.Payload)["id"].(string)
			mac, _ := (*msg.Payload)["arenaserveruuid"].(string)

			onGameStop(
				server.state,
				mac,
				gameid,
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
