package arenaserver

import (
	"errors"

	"github.com/bytearena/bytearena/common/utils/vector"

	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/common/utils"
	uuid "github.com/satori/go.uuid"
)

func (s *Server) RegisterAgent(agentimage, agentname string) {

	///////////////////////////////////////////////////////////////////////////
	// Building the agent entity (gameplay related aspects of the agent)
	///////////////////////////////////////////////////////////////////////////

	arenamap := s.GetGameDescription().GetMapContainer()
	agentSpawnPointIndex := len(s.agentproxies)

	if agentSpawnPointIndex >= len(arenamap.Data.Starts) {
		utils.Debug("arena", "Agent "+agentimage+" cannot spawn, no starting point left")
		return
	}

	agentSpawningPos := arenamap.Data.Starts[agentSpawnPointIndex]

	agententity := s.game.NewEntityAgent(vector.MakeVector2(agentSpawningPos.Point.X, agentSpawningPos.Point.Y))

	///////////////////////////////////////////////////////////////////////////
	// Building the agent proxy (concrete link with container and communication pipe)
	///////////////////////////////////////////////////////////////////////////

	agentproxy := agent.MakeAgentProxyNetwork()
	agentproxy.SetEntityId(agententity.GetID())

	s.setAgentProxy(agentproxy)
	s.agentimages[agentproxy.GetProxyUUID()] = agentimage

	utils.Debug("arena", "Registrer agent "+agentimage)
}

func (s *Server) startAgentContainers() error {

	for _, agentproxy := range s.agentproxies {
		dockerimage := s.agentimages[agentproxy.GetProxyUUID()]

		arenaHostnameForAgents, err := s.containerorchestrator.GetHost()
		if err != nil {
			return errors.New("Failed to fetch arena hostname for agents; " + err.Error())
		}

		container, err := s.containerorchestrator.CreateAgentContainer(agentproxy.GetProxyUUID(), arenaHostnameForAgents, s.port, dockerimage)

		if err != nil {
			return errors.New("Failed to create docker container for " + agentproxy.String() + ": " + err.Error())
		}

		err = s.containerorchestrator.StartAgentContainer(container, s.AddTearDownCall)

		if err != nil {
			return errors.New("Failed to start docker container for " + agentproxy.String() + ": " + err.Error())
		}

		s.AddTearDownCall(func() error {
			s.containerorchestrator.TearDown(container)

			return nil
		})
	}

	return nil
}

func (s *Server) setAgentProxy(agent agent.AgentProxyInterface) {
	s.agentproxiesmutex.Lock()
	defer s.agentproxiesmutex.Unlock()
	s.agentproxies[agent.GetProxyUUID()] = agent
}

func (s *Server) getAgentProxy(agentid string) (agent.AgentProxyInterface, error) {
	var emptyagent agent.AgentProxyInterface

	foundkey, err := uuid.FromString(agentid)
	if err != nil {
		return emptyagent, err
	}

	s.agentproxiesmutex.Lock()
	if foundagent, ok := s.agentproxies[foundkey]; ok {
		s.agentproxiesmutex.Unlock()
		return foundagent, nil
	}
	s.agentproxiesmutex.Unlock()

	return emptyagent, errors.New("Agent" + agentid + " not found")
}
