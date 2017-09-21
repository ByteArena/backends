package arenaserver

import (
	"errors"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	uuid "github.com/satori/go.uuid"
)

func (s *Server) RegisterAgent(agentimage, agentname string) {
	arenamap := s.game.GetMapContainer()
	agentSpawnPointIndex := len(s.agents)

	if agentSpawnPointIndex >= len(arenamap.Data.Starts) {
		utils.Debug("arena", "Agent "+agentimage+" cannot spawn, no starting point left")
		return
	}

	agentSpawningPos := arenamap.Data.Starts[agentSpawnPointIndex]

	agent := agent.MakeNetAgentImp()

	///////////////////////////////////////////////////////////////////////////
	// Building the physical body of the agent
	///////////////////////////////////////////////////////////////////////////

	bodydef := box2d.MakeB2BodyDef()
	bodydef.Position.Set(agentSpawningPos.Point.X, agentSpawningPos.Point.Y)
	bodydef.Type = box2d.B2BodyType.B2_dynamicBody
	bodydef.AllowSleep = false
	bodydef.FixedRotation = true

	body := s.state.PhysicalWorld.CreateBody(&bodydef)

	shape := box2d.MakeB2CircleShape()
	shape.SetRadius(0.5)

	fixturedef := box2d.MakeB2FixtureDef()
	fixturedef.Shape = &shape
	fixturedef.Density = 20.0
	body.CreateFixtureFromDef(&fixturedef)
	body.SetUserData(types.MakePhysicalBodyDescriptor(types.PhysicalBodyDescriptorType.Agent, agent.GetId().String()))
	body.SetBullet(true)

	///////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////////

	agentstate := state.MakeAgentState(agent.GetId(), agentname, body)

	body.SetLinearDamping(agentstate.DragForce * float64(s.tickspersec)) // aerodynamic drag

	s.setAgent(agent)
	s.state.SetAgentState(agent.GetId(), agentstate)

	s.agentimages[agent.GetId()] = agentimage

	utils.Debug("arena", "Registrer agent "+agentimage)
}

func (s *Server) spawnAgents() error {

	for _, agent := range s.agents {
		dockerimage := s.agentimages[agent.GetId()]

		arenaHostnameForAgents, err := s.containerorchestrator.GetHost(&s.containerorchestrator)
		if err != nil {
			return errors.New("Failed to fetch arena hostname for agents; " + err.Error())
		}

		container, err := s.containerorchestrator.CreateAgentContainer(agent.GetId(), arenaHostnameForAgents, s.port, dockerimage)

		if err != nil {
			return errors.New("Failed to create docker container for " + agent.String() + ": " + err.Error())
		}

		err = s.containerorchestrator.StartAgentContainer(container, s.AddTearDownCall)

		if err != nil {
			return errors.New("Failed to start docker container for " + agent.String() + ": " + err.Error())
		}

		s.AddTearDownCall(func() error {
			s.containerorchestrator.TearDown(container)

			return nil
		})
	}

	return nil
}

func (s *Server) setAgent(agent agent.AgentInterface) {
	s.agentsmutex.Lock()
	defer s.agentsmutex.Unlock()

	s.agents[agent.GetId()] = agent
}

func (s *Server) getAgent(agentid string) (agent.AgentInterface, error) {
	var emptyagent agent.AgentInterface

	foundkey, err := uuid.FromString(agentid)
	if err != nil {
		return emptyagent, err
	}

	s.agentsmutex.Lock()
	if foundagent, ok := s.agents[foundkey]; ok {
		s.agentsmutex.Unlock()
		return foundagent, nil
	}
	s.agentsmutex.Unlock()

	return emptyagent, errors.New("Agent" + agentid + " not found")
}
