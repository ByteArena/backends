package state

import (
	"math"
	"math/rand"

	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
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
	Radius             float64
	Mass               float64
	Position           utils.Vector2
	Velocity           utils.Vector2
	Orientation        float64 // heading angle in radian (degree ?) relative to arena north
	MaxSteeringForce   float64 // maximum magnitude the steering force applied to current velocity
	MaxSpeed           float64 // maximum magnitude of the agent velocity
	MaxAngularVelocity float64

	Tag string // attractor

	VisionRadius float64
}

func MakeAgentState() AgentState {
	initialx := rand.Float64() * 800
	initialy := rand.Float64() * 600

	r := 6 + rand.Float64()*6.0

	maxdegreespertick := 8.0 + 5*(1/r) // bigger turn slower

	return AgentState{
		Position:           utils.MakeVector2(initialx, initialy),
		Velocity:           utils.MakeVector2(0, 0),
		MaxSpeed:           5.0,
		MaxSteeringForce:   0.8,
		MaxAngularVelocity: utils.DegreeToRadian(maxdegreespertick), // en radians/tick; Pi = 180°
		Radius:             r,
		Mass:               math.Pi * r * r,
		Tag:                "agent",
		VisionRadius:       300,
	}
}

func (state AgentState) Update() AgentState {
	if state.Velocity.Mag() > 0.00001 {
		state.Orientation = state.Velocity.Angle()
	} else {
		state.Velocity = utils.MakeVector2(0, 0)
	}

	state.Position = state.Position.Add(state.Velocity)

	return state
}

func (state AgentState) mutationSteer(steering utils.Vector2) AgentState {

	// Limit acceleration/deceleration
	velocitymag := state.Velocity.Mag()
	diff := steering.Mag() - velocitymag

	if math.Abs(diff) > state.MaxSteeringForce {
		if diff > 0 {
			steering = steering.SetMag(velocitymag + state.MaxSteeringForce)
		} else {
			steering = steering.SetMag(velocitymag - state.MaxSteeringForce)
		}
	}

	steering = steering.Limit(state.MaxSpeed)
	currentspeed := steering.Mag()

	// angular velocity is relative to current speed
	maxangularvelocity := utils.Map(currentspeed, 0, state.MaxSpeed, state.MaxAngularVelocity*0.05, state.MaxAngularVelocity)

	abssteering := state.localAngleToAbsoluteAngleVec(steering, &maxangularvelocity)
	state.Velocity = abssteering.Limit(state.MaxSpeed)

	return state
}

func (state AgentState) mutationShoot(serverstate *ServerState, aiming utils.Vector2) AgentState {

	// on passe le vecteur de visée d'un angle relatif à un angle absolu
	absaiming := state.localAngleToAbsoluteAngleVec(aiming, nil)

	projectile := ProjectileState{
		Position: state.Position.Clone(),
		Velocity: state.Position.Add(absaiming), // adding the agent position to "absolutize" the target vector
		From:     state,
		Ttl:      1,
	}

	projectileid := uuid.NewV4()

	serverstate.Projectilesmutex.Lock()
	serverstate.Projectiles[projectileid] = projectile
	serverstate.Projectilesmutex.Unlock()

	return state
}

func (state AgentState) localAngleToAbsoluteAngleVec(vec utils.Vector2, maxangleconstraint *float64) utils.Vector2 {

	abscurrentagentangle := state.Orientation
	absvecangle := vec.Angle()

	relvecangle := absvecangle

	// On passe de 0° / 360° à -180° / +180°
	if absvecangle > math.Pi { // 180° en radians
		relvecangle -= math.Pi * 2 // 360° en radian
	}

	// On contraint la vélocité angulaire à un maximum
	if maxangleconstraint != nil {
		maxangleconstraintval := *maxangleconstraint
		if math.Abs(relvecangle) > maxangleconstraintval {
			if relvecangle > 0 {
				relvecangle = maxangleconstraintval
			} else {
				relvecangle = -1 * maxangleconstraintval
			}
		}
	}

	return vec.SetAngle(abscurrentagentangle + relvecangle)
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
