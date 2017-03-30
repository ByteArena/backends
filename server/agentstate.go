package main

type AgentState struct {
	Position *Vector2
	Velocity *Vector2
	Radius   float64
}

func NewAgentState() *AgentState {
	return &AgentState{
		Position: NewVector2(0, 0),
		Velocity: NewVector2(0, 0),
	}
}

func (state *AgentState) update() {
	state.Position.add(state.Velocity)
}

func (state *AgentState) mutationSteer(v *Vector2) {
	state.Velocity.add(v)
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
