package arenamaster

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"

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
			mac, _ := (*msg.Payload)["arenaserveruuid"].(string)
			log.Println(mac)
			id, _ := strconv.Atoi(strings.Split(mac, ":")[0])

			server.state.UpdateStateAddIdleArena(id)

			// onGameHandshake(
			// 	server.state,
			// 	msg.Payload,
			// )

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
