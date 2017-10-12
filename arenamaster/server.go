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
	vmid "github.com/bytearena/schnapps/id"
	vmmeta "github.com/bytearena/schnapps/metadata"
	vmscheduler "github.com/bytearena/schnapps/scheduler"
	vmtypes "github.com/bytearena/schnapps/types"
)

var (
	inc     = 0
	dnsZone = "bytearena.com."
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

func (server *Server) createScheduler() *vmscheduler.Pool {
	provisionVmFn := func() *vm.VM {
		inc++
		id := inc

		vm, err := server.SpawnArena(id)
		server.state.UpdateStateAddBootingVM(id, vm)

		if err != nil {
			utils.RecoverableError("vm", "Could not start ("+strconv.Itoa(id)+"): "+err.Error())
			server.state.UpdateStateVMErrored(id)

			return nil
		} else {
			err := vm.WaitUntilBooted()

			if err != nil {
				utils.RecoverableError("vm", "Could not wait until VM is booted")
				server.state.UpdateStateVMErrored(id)
			} else {
				server.state.UpdateStateVMBooted(id)
				utils.Debug("vm", "VM ("+strconv.Itoa(id)+") booted")
			}

			return vm
		}
	}

	healtcheckVmFn := func(vm *vm.VM) bool {
		// blockChan := make(chan bool)
		// mac := vmid.GetVMMAC(vm)

		// go func() {
		// 	server.mqclient.Publish("game", mac+".healthcheck", types.MQPayload{})
		// }()

		// <-blockChan

		// return false
		if vm == nil {
			return false
		} else {
			return true
		}
	}

	pool, schedulerErr := vmscheduler.NewFixedVMPool(3, provisionVmFn, healtcheckVmFn)

	if schedulerErr != nil {
		panic(schedulerErr)
	}

	return pool
}

func (server *Server) createDNSServer() {

	dnsRecords := map[string]string{
		"static." + dnsZone:       server.vmBridgeIP,
		"redis.net." + dnsZone:    server.vmBridgeIP,
		"graphql.net." + dnsZone:  server.vmBridgeIP,
		"registry.net." + dnsZone: server.vmBridgeIP,
	}

	DNSServer := vmdns.MakeServer(server.vmBridgeIP+":53", dnsZone, dnsRecords)

	// DNSServer.SetOnRequestHook(func(addr string) {
	// 	utils.Debug("dns-server", "query for "+addr)
	// })

	go func() {
		err := DNSServer.Start()
		utils.Check(err, "Could not start DNS server")

		server.DNSServer = &DNSServer
	}()
}

func (server *Server) createMetadataServer() {
	retrieveVMFn := func(id string) *vm.VM {
		vm := FindVMByMAC(server.state, id)

		return vm
	}

	metadataServer := vmmeta.NewServer(server.vmBridgeIP+":8080", retrieveVMFn)

	go func() {
		err := metadataServer.Start()
		utils.Check(err, "Could not start metadata server")

		server.MetadataServer = metadataServer
	}()
}

func (server *Server) createDHCPServer() {
	var err error
	cidr := server.vmSubnet

	server.DHCPServer, err = vmdhcp.NewDHCPServer(cidr)
	utils.Check(err, "Could not create DHCP server")
}

func (server *Server) Run() {
	listener := MakeListener(server.brokerclient)

	server.createDHCPServer()
	server.createDNSServer()
	server.createMetadataServer()

	pool := server.createScheduler()
	utils.Debug("vm", "Scheduler running and initialized")

	for {
		select {
		case <-server.stopChan:
			return

		case <-listener.arenaAdd:
			utils.Debug("err", "implement this")

		case msg := <-listener.arenaHalt:
			go func() {
				id, _ := strconv.Atoi((*msg.Payload)["id"].(string))

				if data := server.state.QueryState(id, state.STATE_RUNNING_VM); data != nil {
					server.state.UpdateStateVMHalted(id)

					runningVM := data.(*vm.VM)
					runningVM.Quit()

					err := pool.Delete(runningVM)

					if err != nil {
						utils.RecoverableError("vm", "Could not halt ("+strconv.Itoa(id)+"): "+err.Error())
					}
				} else {
					utils.RecoverableError("vm", "Could not halt ("+strconv.Itoa(id)+"): VM is not running")
				}
			}()

		case msg := <-listener.gameLaunch:
			go func() {
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

				vm, err := pool.SelectAndPop(func(vm *vm.VM) bool {
					return server.state.GetStatus(vm.Config.Id)&state.STATE_IDLE_ARENA != 0
				})

				if vm != nil && err == nil {
					server.state.UpdateStateTriedLaunchArena(vm.Config.Id)

					onGameLaunch(
						gameid,
						server.brokerclient,
						server.graphqlclient,
						vm,
					)

					// FIXME(sven): let's assume it has been launched for now
					server.state.UpdateStateConfirmedLaunchArena(vm.Config.Id)
				} else if vm == nil {
					utils.RecoverableError("vm", "Could not launch game: no arena available")

					// Retry in 1min
					<-time.After(1 * time.Minute)
					listener.gameLaunch <- msg
				} else {
					utils.RecoverableError("vm", "Could not launch game: "+err.Error())

					pool.Release(vm)

					// Retry in 30sec
					<-time.After(30 * time.Second)
					listener.gameLaunch <- msg
				}
			}()

		case msg := <-listener.gameLaunched:
			go func() {
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

			}()
		case msg := <-listener.gameHandshake:
			go func() {
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
			}()

		case msg := <-listener.gameStopped:
			go func() {
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
			}()

		case <-listener.debugGetVMStatus:
			go handleDebugGetVMStatus(server.brokerclient, server.state)
		}
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

func (server *Server) SpawnArena(id int) (*vm.VM, error) {
	mac := vmid.GenerateRandomMAC()
	ip, ipErr := server.DHCPServer.Pop()

	if ipErr != nil {
		return nil, ipErr
	}

	meta := vmtypes.VMMetadata{
		"IP": ip,
	}

	config := vmtypes.VMConfig{
		NICs: []interface{}{
			vmtypes.NICBridge{
				Bridge: server.vmBridgeName,
				MAC:    mac,
			},
		},
		Id:            id,
		MegMemory:     2048,
		CPUAmount:     1,
		CPUCoreAmount: 1,
		ImageLocation: server.vmRawImageLocation,
		Metadata:      meta,
	}

	arenaVm := vm.NewVM(config)

	startErr := arenaVm.Start()

	if startErr != nil {
		return nil, startErr
	}

	utils.Debug("vm", "Started new VM ("+mac+")")

	return arenaVm, nil
}
