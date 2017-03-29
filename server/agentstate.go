package main

import "math"

type AgentState struct {
	Counter      int
	Position     Vector2
	Velocity     Vector2
	Acceleration Vector2
	Radius       float64
}

func (state *AgentState) applyPhysics() {
	state.Velocity.add(state.Acceleration)
	state.Position.add(state.Velocity)
}

func (state *AgentState) mutationIncrement() {
	state.Counter++
}

func (state *AgentState) mutationAccelerate(v Vector2) {
	state.Velocity.add(v)
}

func (state *AgentState) clone() *AgentState {
	clone := *state
	return &clone
}

func (state *AgentState) validate() bool {
	return state.Counter >= 0
}

func (state *AgentState) validateTransition(fromstate *AgentState) bool {
	return math.Abs(float64(state.Counter-fromstate.Counter)) <= 2
}
