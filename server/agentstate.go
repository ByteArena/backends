package server

import (
	"math/rand"

	"github.com/netgusto/bytearena/utils"
)

type AgentState struct {
	Position *utils.Vector2
	Velocity *utils.Vector2
	Radius   float64
}

func NewAgentState() *AgentState {
	initialx := rand.Float64() * 800
	initialy := rand.Float64() * 600

	return &AgentState{
		Position: utils.NewVector2(initialx, initialy),
		Velocity: utils.NewVector2(0, 0),
	}
}

func (state *AgentState) update() {
	state.Position.Add(state.Velocity)
}

func (state *AgentState) mutationSteer(v *utils.Vector2) {
	state.Velocity.Add(v)
}

func (state *AgentState) clone() *AgentState {
	clone := *state
	return &clone
}

func (state *AgentState) validate() bool {
	//return state.Counter >= 0
	return true
}

func (state *AgentState) validateTransition(fromstate *AgentState) bool {
	//return math.Abs(float64(state.Counter-fromstate.Counter)) <= 2
	return true
}
