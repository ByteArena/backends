package arenamaster

import (
	"encoding/json"
	"strconv"
	"time"

	arenamasterGraphql "github.com/bytearena/bytearena/arenamaster/graphql"
	"github.com/bytearena/bytearena/arenamaster/state"
	"github.com/bytearena/bytearena/common/graphql"
	"github.com/bytearena/bytearena/common/influxdb"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/schnapps"
	vmdhcp "github.com/bytearena/schnapps/dhcp"
	vmdns "github.com/bytearena/schnapps/dns"
	vmmeta "github.com/bytearena/schnapps/metadata"
)

var (
	EVENT_COUNTER = influxdb.NewCounter()

	TIME_BETWEEN_VM_RUNNING_AND_ARENA_IDLE = 1 * time.Minute
)

type ListeningChanStruct chan bool
type Server struct {
	stopChan           ListeningChanStruct
	brokerclient       *mq.Client
	graphqlclient      *graphql.Client
	state              *state.State
	influxdbClient     *influxdb.Client
	DNSServer          *vmdns.Server
	MetadataServer     *vmmeta.MetadataHTTPServer
	DHCPServer         *vmdhcp.DHCPServer
	vmRawImageLocation string
	vmBridgeName       string
	vmBridgeIP         string
	vmSubnet           string
}

func NewServer(mq *mq.Client, gql *graphql.Client, vmRawImageLocation, vmBridgeName, vmBridgeIP, vmSubnet string) *Server {
	stopChan := make(ListeningChanStruct)

	influxdbClient, influxdbClientErr := influxdb.NewClient("arenamaster")
	utils.Check(influxdbClientErr, "Unable to create influxdb client")

	s := &Server{
		brokerclient:       mq,
		graphqlclient:      gql,
		state:              state.NewState(),
		stopChan:           stopChan,
		influxdbClient:     influxdbClient,
		vmRawImageLocation: vmRawImageLocation,
		vmBridgeName:       vmBridgeName,
		vmBridgeIP:         vmBridgeIP,
		vmSubnet:           vmSubnet,
	}

	err := s.startStateReporting()

	utils.CheckWithFunc(err, func() string {
		panic("Could not start state reporting: " + err.Error())
	})

	return s
}

func (server *Server) startStateReporting() error {

	server.influxdbClient.Loop(func() {
		fields := make(map[string]interface{})

		// Transform map[string]int into map[string]interface{}
		// it works somehow
		for k, v := range server.state.DebugGetStateDistribution() {
			fields[k] = v
		}

		fields["events-per-period"] = EVENT_COUNTER.GetAndReset()

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

	server.createDHCPServer()
	server.createDNSServer()
	server.createMetadataServer()

	healthchecks := NewArenaHealthcheck(listener.gameHealthcheckRes, server.brokerclient)

	pool := server.createScheduler(listener, healthchecks)
	utils.Debug("vm", "Scheduler running and initialized")

	for {
		select {
		case <-server.stopChan:
			return

		case msg := <-listener.arenaHalt:
			{
				id, _ := strconv.Atoi((*msg.Payload)["id"].(string))

				if data := server.state.QueryState(id, state.STATE_RUNNING_VM); data != nil {
					runningVM := data.(*vm.VM)
					quitErr := runningVM.Quit()

					if quitErr != nil {
						utils.RecoverableError("vm", "Could not quit ("+strconv.Itoa(id)+"): "+quitErr.Error())
					}

					err := pool.Delete(runningVM)

					if err != nil {
						utils.RecoverableError("vm", "Could not halt ("+strconv.Itoa(id)+"): "+err.Error())
					}

					server.state.UpdateStateVMHalted(id)
				} else {
					utils.RecoverableError("vm", "Could not halt ("+strconv.Itoa(id)+"): VM is not running")
				}
			}

		case msg := <-listener.gameLaunch:
			{
				gameid, _ := (*msg.Payload)["id"].(string)

				// Check if the gameid isn't running already
				var isGameAlreadyRunning bool
				server.state.Map(func(element *state.DataContainer) {
					if isGameAlreadyRunning == true {
						return
					}

					vm := element.Data.(*vm.VM)
					isRunning := element.Status&state.STATE_RUNNING_ARENA != 0

					vmGameId, hasVmGameId := vm.Config.Metadata["gameid"]

					if isRunning && hasVmGameId && vmGameId == gameid {
						isGameAlreadyRunning = true
					}
				})

				if isGameAlreadyRunning == true {
					utils.RecoverableError("vm", "Could not launch game: Game is already running")
					return
				}

				vm, err := pool.Pop()

				if vm != nil && err == nil {
					server.state.UpdateStateTriedLaunchArena(vm.Config.Id)

					onGameLaunch(
						gameid,
						server.brokerclient,
						server.graphqlclient,
						vm,
					)
				} else if vm == nil {
					utils.RecoverableError("vm", "Could not launch game: no arena available")
				} else {
					utils.RecoverableError("vm", "Could not launch game: "+err.Error())

					err := pool.Release(vm)

					if err != nil {
						utils.RecoverableError("vm", "Could not release ("+strconv.Itoa(vm.Config.Id)+"): "+err.Error())
					}

					go func() {
						// Retry in 30sec
						<-time.After(30 * time.Second)
						listener.gameLaunch <- msg
					}()
				}
			}

		case msg := <-listener.gameLaunched:
			{
				mac, _ := (*msg.Payload)["arenaserveruuid"].(string)
				gameid, _ := (*msg.Payload)["id"].(string)
				vm := FindVMByMAC(server.state, mac)

				if vm != nil {
					server.state.UpdateStateConfirmedLaunchArena(vm.Config.Id)

					arenamasterGraphql.ReportGameLaunched(gameid, mac, server.graphqlclient)
					utils.Debug("master", mac+" launched")

				} else {
					utils.RecoverableError("game-launched", "VM with MAC ("+mac+") does not exists")
				}
			}

		case msg := <-listener.gameHandshake:
			{
				mac, _ := (*msg.Payload)["arenaserveruuid"].(string)
				vm := FindVMByMAC(server.state, mac)

				// Refuse handshakes from already running arenas
				if server.state.GetStatus(vm.Config.Id)&state.STATE_RUNNING_ARENA != 0 {
					return
				}

				if vm != nil {
					server.state.UpdateStateAddIdleArena(vm.Config.Id)
					utils.Debug("master", mac+" joined")
				} else {
					utils.RecoverableError("game-handshake", "VM with MAC ("+mac+") does not exists")
				}
			}

		case msg := <-listener.gameStopped:
			{
				gameid, _ := (*msg.Payload)["id"].(string)
				mac, _ := (*msg.Payload)["arenaserveruuid"].(string)

				vm := FindVMByMAC(server.state, mac)

				if vm != nil {
					id := vm.Config.Id
					server.state.UpdateStateStoppedArena(id)

					arenamasterGraphql.ReportGameStopped(
						server.state,
						mac,
						gameid,
						server.graphqlclient,
					)

					haltMsg := types.NewMQMessage(
						"arena-master",
						"halt",
					).SetPayload(types.MQPayload{
						"id": strconv.Itoa(id),
					})

					listener.arenaHalt <- *haltMsg
				} else {
					utils.RecoverableError("game-stopped", "VM with MAC ("+mac+") does not exists")
				}
			}

		case <-listener.debugGetVMStatus:
			go handleDebugGetVMStatus(server.brokerclient, server.state, healthchecks)
		}

		EVENT_COUNTER.Add(1)
	}
}

func (server *Server) Stop() {
	server.stopChan <- true
	server.influxdbClient.TearDown()

	if server.DNSServer != nil {
		server.DNSServer.Stop()
	}

	if server.MetadataServer != nil {
		server.MetadataServer.Stop()
	}

	close(server.stopChan)
}
