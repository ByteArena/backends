package agent

import (
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
)

type Agent interface {
	GetId() uuid.UUID
	String() string
	GetPerception(swarmstate *state.ServerState) state.Perception
	GetState(swarmstate *state.ServerState) state.AgentState
	SetState(swarmstate *state.ServerState, state state.AgentState)
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

func (agent AgentImp) GetPerception(swarmstate *state.ServerState) state.Perception {
	p := state.Perception{}
	agentstate := agent.GetState(swarmstate)

	p.Internal.Velocity = agentstate.Velocity.Clone()
	p.Internal.Proprioception = agentstate.Radius

	// On rend la position de l'attractor relative à l'agent
	p.Objective.Attractor = swarmstate.Pin.Clone().Sub(agentstate.Position)

	p.Specs.MaxSpeed = agentstate.MaxSpeed
	p.Specs.MaxSteeringForce = agentstate.MaxSteeringForce

	return p
}

func (agent AgentImp) GetState(swarmstate *state.ServerState) state.AgentState {
	return swarmstate.Agents[agent.GetId()]
}

func (agent AgentImp) SetState(swarmstate *state.ServerState, state state.AgentState) {
	swarmstate.Agents[agent.GetId()] = state
}

func (agent AgentImp) GetTickedChan() chan utils.Tickturn {
	return agent.tickedchan
}
