package state

import (
	"math/rand"

	"github.com/netgusto/bytearena/utils"
)

// Agent is a Simple Vehicle Model from Reynolds (http://www.red3d.com/cwr/steer/gdc99/)
// Agent Physics is based on forward Euler integration
//
// At each simulation step, behaviorally determined steering forces (as limited by max_force)
// are applied to the vehicle’s point mass. This produces an acceleration equal to the steering force
// divided by the vehicle’s mass. That acceleration is added to the old velocity to produce
// a new velocity, which is then truncated by max_speed.
// Finally, the velocity is added to the old position:
//
// steering_force = limitmagnitude(steering_direction, max_force)
// acceleration = steering_force / mass
// velocity = limitmagnitude(velocity + acceleration, max_speed)
// position = position + velocity
//
// Because of its assumption of velocity alignment, this simple vehicle model cannot simulate
// effects such as skids, spins or slides. Furthermore this model allows the vehicle to turn
// when its speed is zero. Most real vehicles cannot do this (they are “non-holonomic”)
// and in any case it allows undesirably large changes in orientation during a single time step.
//
// This problem can be solved by placing an additional constraint on change of orientation,
// or by limiting the lateral steering component at low speeds,
// or by simulating moment of inertia.

type AgentState struct {
	Radius           float64
	Mass             float64
	Position         utils.Vector2
	Velocity         utils.Vector2
	MaxSteeringForce float64 // maximum magnitude the steering force applied to current velocity
	MaxSpeed         float64 // maximum magnitude of the agent velocity
	Orientation      float64 // heading angle in radian (degree ?)
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
