package state

import (
	"math/rand"

	"github.com/netgusto/bytearena/utils"
)

type AgentState struct {
	Position         utils.Vector2
	Velocity         utils.Vector2
	Radius           float64
	MaxSpeed         float64 // maximum magnitude of the agent velocity
	MaxSteeringForce float64 // maximum magnitude the steering force applied to current velocity
}

func MakeAgentState() AgentState {
	initialx := rand.Float64() * 800
	initialy := rand.Float64() * 600

	return AgentState{
		Position:         utils.MakeVector2(initialx, initialy),
		Velocity:         utils.MakeVector2(0, 0),
		MaxSpeed:         8.0,
		MaxSteeringForce: 4.0,
		Radius:           8.0,
	}
}

func (state AgentState) Update() AgentState {
	state.Position = state.Position.Add(state.Velocity)
	return state
}

func (state AgentState) mutationSteer(v utils.Vector2) AgentState {
	steeringvec := v.Limit(state.MaxSteeringForce)
	state.Velocity = state.Velocity.Add(steeringvec).Limit(state.MaxSpeed)
	return state
}

func (state AgentState) clone() AgentState {
	return state // yes, passed by value !
}

func (state AgentState) validate() bool {
	return true
}

func (state AgentState) validateTransition(fromstate AgentState) bool {
	return true
}
