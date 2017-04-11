package agent

import (
	"github.com/netgusto/bytearena/server/protocol"
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
)

type Agent interface {
	GetId() uuid.UUID
	String() string
	GetPerception(serverstate *state.ServerState) state.Perception
	SetPerception(perception state.Perception, comm protocol.AgentCommunicator, agentstate state.AgentState) // abstract method
}

type AgentImp struct {
	id uuid.UUID
	//tickedchan chan utils.Tickturn
}

func MakeAgentImp() AgentImp {
	return AgentImp{
		id: uuid.NewV4(), // random uuid
	}
}

func (agent AgentImp) GetId() uuid.UUID {
	return agent.id
}

func (agent AgentImp) String() string {
	return "<AgentImp(" + agent.GetId().String() + ")>"
}

func (agent AgentImp) GetPerception(serverstate *state.ServerState) state.Perception {
	p := state.Perception{}
	agentstate := serverstate.GetAgentState(agent.GetId())

	p.Internal.Velocity = agentstate.Velocity.Clone()
	p.Internal.Proprioception = agentstate.Radius

	// On trouve l'agent déclaré comme attractor
	// TODO: troiuver mieux qu'une itération du Map à chaque GetPerception
	serverstate.Agentsmutex.Lock()
	var attractorpos = utils.MakeVector2(0, 0)
	for _, agentstate := range serverstate.Agents {
		if agentstate.Tag == "attractor" {
			attractorpos = agentstate.Position
		}
	}
	serverstate.Agentsmutex.Unlock()

	// On rend la position de l'attractor relative à l'agent
	p.Objective.Attractor = attractorpos.Sub(agentstate.Position)

	p.Specs.MaxSpeed = agentstate.MaxSpeed
	p.Specs.MaxSteeringForce = agentstate.MaxSteeringForce

	return p
}

func (agent AgentImp) SetPerception(perception state.Perception, comm protocol.AgentCommunicator, agentstate state.AgentState) {
	// I'm abstract, override me !
}
