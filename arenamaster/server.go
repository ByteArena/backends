package arenamaster

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/bytearena/schnapps"
	vmdhcp "github.com/bytearena/schnapps/dhcp"
	vmdns "github.com/bytearena/schnapps/dns"
	vmmeta "github.com/bytearena/schnapps/metadata"

	"github.com/xtuc/schaloop"

	arenamasterGraphql "github.com/bytearena/backends/arenamaster/graphql"
	"github.com/bytearena/backends/arenamaster/state"
	"github.com/bytearena/backends/common/graphql"
	"github.com/bytearena/backends/common/influxdb"
	"github.com/bytearena/backends/common/mq"

	bamq "github.com/bytearena/core/common/mq"
	"github.com/bytearena/core/common/types"
	"github.com/bytearena/core/common/utils"
)

var (
	EVENT_COUNTER = influxdb.NewCounter()

	TIME_BETWEEN_VM_RUNNING_AND_ARENA_IDLE = 1 * time.Minute
)

type Server struct {
	stopChan           chan bool
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
	stopChan := make(chan bool)

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

func unmarshalMQMessage(msg bamq.BrokerMessage) (error, *types.MQMessage) {
	var message types.MQMessage
	err := json.Unmarshal(msg.Data, &message)
	if err != nil {
		return err, nil
	} else {
		return nil, &message
	}
}

func resToGeneric(old Res) chan interface{} {
	new := make(chan interface{})

	go func() {
		for {
			v := <-old
			new <- v
		}
	}()

	return new
}

func boolToGeneric(old chan bool) chan interface{} {
	new := make(chan interface{})

	go func() {
		for {
			v := <-old
			new <- v
		}
	}()

	return new
}

func (server *Server) Run() {
	waitChan := make(chan bool)
	listener := MakeListener(server.brokerclient)

	eventloop := schaloop.NewEventLoop()
	eventloop.StartWithTimeout(time.Duration(2 * time.Minute))

	server.createDHCPServer()
	server.createDNSServer()
	server.createMetadataServer()

	healthchecks := NewArenaHealthcheck(listener.gameHealthcheckRes, server.brokerclient)

	pool, waitUntilReady := server.createScheduler(eventloop, listener, healthchecks)
	utils.Debug("vm", "Scheduler running and initialized")

	<-waitUntilReady
	healthchecks.StartChecks(eventloop)

	eventloop.QueueWorkFromChannel("arena-halt", resToGeneric(listener.arenaHalt), func(data interface{}) {
		msg := data.(types.MQMessage)
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
	})

	eventloop.QueueWorkFromChannel("game-launch", resToGeneric(listener.gameLaunch), func(data interface{}) {
		msg := data.(types.MQMessage)
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
	})

	eventloop.QueueWorkFromChannel("game-launched", resToGeneric(listener.gameLaunched), func(data interface{}) {
		msg := data.(types.MQMessage)
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
	})

	eventloop.QueueWorkFromChannel("game-handshake", resToGeneric(listener.gameHandshake), func(data interface{}) {
		msg := data.(types.MQMessage)
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
	})

	eventloop.QueueWorkFromChannel("game-stopped", resToGeneric(listener.gameStopped), func(data interface{}) {
		msg := data.(types.MQMessage)
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

			go func() {
				listener.arenaHalt <- *haltMsg
			}()
		} else {
			utils.RecoverableError("game-stopped", "VM with MAC ("+mac+") does not exists")
		}
	})

	eventloop.QueueWorkFromChannel("debug-getvmstatus", resToGeneric(listener.debugGetVMStatus), func(data interface{}) {
		go handleDebugGetVMStatus(server.brokerclient, server.state, healthchecks)
	})

	eventloop.QueueWorkFromChannel("stop", boolToGeneric(server.stopChan), func(data interface{}) {
		waitChan <- false
	})

	<-waitChan
	eventloop.Stop()
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
