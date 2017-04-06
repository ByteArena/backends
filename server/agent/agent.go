package agent

import (
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
)

type Agent interface {
	GetId() uuid.UUID
	String() string
	GetPerception(serverstate *state.ServerState) state.Perception
	GetTickedChan() chan utils.Tickturn
}

type AgentImp struct {
	id         uuid.UUID
	tickedchan chan utils.Tickturn
}

func MakeAgentImp() AgentImp {
	return AgentImp{
		id:         uuid.NewV4(),                  // random uuid
		tickedchan: make(chan utils.Tickturn, 10), // can buffer up to 10 turns, to avoid blocking
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

	// On rend la position de l'attractor relative à l'agent
	p.Objective.Attractor = serverstate.Pin.Clone().Sub(agentstate.Position)

	p.Specs.MaxSpeed = agentstate.MaxSpeed
	p.Specs.MaxSteeringForce = agentstate.MaxSteeringForce

	return p
}

func (agent AgentImp) GetTickedChan() chan utils.Tickturn {
	return agent.tickedchan
}
