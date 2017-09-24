package arenaserver

import (
	"encoding/json"
	"errors"
	"runtime"
	"strconv"
	"time"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils"
)

func (server *Server) Start() (chan interface{}, error) {

	utils.Debug("arena", "Listen")
	block := server.listen()

	utils.Debug("arena", "Starting agent containers")
	err := server.startAgentContainers()

	if err != nil {
		return nil, errors.New("Failed to start agent containers: " + err.Error())
	}

	server.AddTearDownCall(func() error {
		utils.Debug("arena", "Publish game state ("+server.arenaServerUUID+"stopped)")

		game := server.GetGameDescription()

		err := server.mqClient.Publish("game", "stopped", types.NewMQMessage(
			"arena-server",
			"Arena Server "+server.arenaServerUUID+", game "+game.GetId()+" stopped",
		).SetPayload(types.MQPayload{
			"id":              game.GetId(),
			"arenaserveruuid": server.arenaServerUUID,
		}))

		return err
	})

	return block, nil
}

func (server *Server) Stop() {
	utils.Debug("arena-server", "TearDown from stop")
	server.TearDown()
}

func (s *Server) SubscribeStateObservation() chan interface{} {
	ch := make(chan interface{})
	s.stateobservers = append(s.stateobservers, ch)
	return ch
}

func (s *Server) SendLaunched() {
	payload := types.MQPayload{
		"id":              s.GetGameDescription().GetId(),
		"arenaserveruuid": s.arenaServerUUID,
	}

	s.mqClient.Publish("game", "launched", types.NewMQMessage(
		"arena-server",
		"Arena Server "+s.arenaServerUUID+" launched",
	).SetPayload(payload))

	payloadJson, _ := json.Marshal(payload)

	utils.Debug("arena-server", "Send game launched: "+string(payloadJson))
}

func (server *Server) onAgentsReady() {
	utils.Debug("arena", "Agents are ready; starting in 1 second")
	time.Sleep(time.Duration(time.Second * 1))

	go func() {
		stopChannel := make(chan bool)
		server.monitoring(stopChannel)

		server.AddTearDownCall(func() error {
			stopChannel <- true
			return nil
		})
	}()

	server.startTicking()
}

func (server *Server) startTicking() {

	tickduration := time.Duration((1000000 / time.Duration(server.tickspersec)) * time.Microsecond)
	ticker := time.Tick(tickduration)

	server.AddTearDownCall(func() error {
		server.stopticking <- true
		close(server.stopticking)

		return nil
	})

	go func() {

		for {
			select {
			case <-server.stopticking:
				{
					utils.Debug("core-loop", "Received stop ticking signal")
					notify.Post("app:stopticking", nil)
					break
				}
			case <-ticker:
				{
					server.doTick()
				}
			}
		}
	}()
}

func (server *Server) doTick() {

	turn := server.getTurn()
	server.setTurn(turn.Next())

	dolog := (turn.GetSeq() % server.tickspersec) == 0

	if dolog {
		utils.Debug("core-loop", "######## Tick ######## "+strconv.Itoa(turn.GetSeq()))
	}

	///////////////////////////////////////////////////////////////////////////
	// Updating world state
	///////////////////////////////////////////////////////////////////////////
	server.update()

	///////////////////////////////////////////////////////////////////////////
	// Refreshing perception for every agent
	///////////////////////////////////////////////////////////////////////////

	arenamap := server.GetGameDescription().GetMapContainer()
	for _, agentproxy := range server.agentproxies {
		go func(server *Server, agentproxy agent.AgentProxyInterface, arenamap *mapcontainer.MapContainer) {

			err := agentproxy.SetPerception(
				server.GetGame().ComputeAgentPerception(arenamap, agentproxy.GetEntityId()),
				server,
			)
			if err != nil {
				utils.Debug("arenaserver", "ERROR: could not set perception on agent "+agentproxy.GetProxyUUID().String())
			}

		}(server, agentproxy, arenamap)
	}

	///////////////////////////////////////////////////////////////////////////
	// Pushing updated state to viz
	///////////////////////////////////////////////////////////////////////////
	for _, subscriber := range server.stateobservers {
		go func(s chan interface{}) {
			s <- nil
		}(subscriber)
	}

	///////////////////////////////////////////////////////////////////////////

	if dolog {
		// Debug : Nombre de goroutines
		utils.Debug("core-loop", "Goroutines in flight : "+strconv.Itoa(runtime.NumGoroutine()))
	}
}

func (s *Server) AddTearDownCall(fn types.TearDownCallback) {
	s.tearDownCallbacksMutex.Lock()
	defer s.tearDownCallbacksMutex.Unlock()

	s.tearDownCallbacks = append(s.tearDownCallbacks, fn)
}

func (server *Server) TearDown() {
	utils.Debug("arena", "teardown")
	server.containerorchestrator.TearDownAll()

	server.tearDownCallbacksMutex.Lock()

	for i := len(server.tearDownCallbacks) - 1; i >= 0; i-- {
		utils.Debug("teardown", "Executing TearDownCallback")
		server.tearDownCallbacks[i]()
	}

	// Reset to avoid calling teardown callback multiple times
	server.tearDownCallbacks = make([]types.TearDownCallback, 0)

	server.tearDownCallbacksMutex.Unlock()
}