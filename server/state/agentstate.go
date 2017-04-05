package state

import (
	"math/rand"

	"github.com/netgusto/bytearena/utils"
)

type ProjectileState struct {
	Position utils.Vector2
	Velocity utils.Vector2
	Ttl      int
	Radius   float64
	From     AgentState
}

type AgentState struct {
	Position utils.Vector2
	Velocity utils.Vector2
	Radius   float64
}

func MakeAgentState() AgentState {
	initialx := rand.Float64() * 800
	initialy := rand.Float64() * 600

	return AgentState{
		Position: utils.MakeVector2(initialx, initialy),
		Velocity: utils.MakeVector2(0, 0),
	}
}

func (state AgentState) Update() AgentState {
	state.Position = state.Position.Add(state.Velocity)
	return state
}

func (state AgentState) mutationSteer(v utils.Vector2) AgentState {
	state.Velocity = state.Velocity.Add(v)
	return state
}

func (state AgentState) clone() AgentState {
	return state // yes, passed by value !
}

func (state AgentState) validate() bool {
	//return state.Counter >= 0
	return true
}

func (state AgentState) validateTransition(fromstate AgentState) bool {
	//return math.Abs(float64(state.Counter-fromstate.Counter)) <= 2
	return true
}
