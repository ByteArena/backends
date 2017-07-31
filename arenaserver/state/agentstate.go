package state

import (
	"math"

	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils/number"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
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
	Position           vector.Vector2
	Velocity           vector.Vector2
	Orientation        float64 // heading angle in radian (degree ?) relative to arena north
	MaxSteeringForce   float64 // maximum magnitude the steering force applied to current velocity
	MaxSpeed           float64 // maximum magnitude of the agent velocity
	MaxAngularVelocity float64

	Tag string // attractor

	VisionRadius float64 // radius of vision circle
	VisionAngle  float64 // angle of FOV
}

func MakeAgentState(start mapcontainer.MapStart) AgentState {
	initialx := start.Point.X
	initialy := start.Point.Y

	r := 0.1

	return AgentState{
		Position: vector.MakeVector2(initialx, initialy),
		//Velocity:           vector.MakeVector2(0.00001, 1),
		MaxSpeed:           1.0,
		MaxSteeringForce:   1.5,
		MaxAngularVelocity: number.DegreeToRadian(18), // en radians/tick; Pi = 180°
		Radius:             r,
		Mass:               math.Pi * r * r,
		Tag:                "agent",
		VisionRadius:       40,
		VisionAngle:        number.DegreeToRadian(180),
	}
}

func (state AgentState) Update() AgentState {
	//newPosition := state.Position.Add(state.Velocity)
	//x, y := newPosition.Get()
	// if x < 0 || y < 0 || x > 1000 || y > 1000 {
	// 	// nothing
	// } else {
	// 	state.Position = state.Position.Add(state.Velocity)
	// }

	state.Position = state.Position.Add(state.Velocity)
	state.Orientation = state.Velocity.Angle()
	return state
}

func (state AgentState) mutationSteer(steering vector.Vector2) AgentState {

	prevmag := state.Velocity.Mag()
	diff := steering.Mag() - prevmag
	if math.Abs(diff) > state.MaxSteeringForce {
		if diff > 0 {
			steering = steering.SetMag(prevmag + state.MaxSteeringForce)
		} else {
			steering = steering.SetMag(prevmag - state.MaxSteeringForce)
		}
	}
	abssteering := state.localAngleToAbsoluteAngleVec(steering, &state.MaxAngularVelocity)
	state.Velocity = abssteering.Limit(state.MaxSpeed)

	return state
}

func (state AgentState) mutationShoot(serverstate *ServerState, aiming vector.Vector2) AgentState {

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

func (state AgentState) localAngleToAbsoluteAngleVec(vec vector.Vector2, maxangleconstraint *float64) vector.Vector2 {

	abscurrentagentangle := state.Orientation
	absvecangle := vec.Angle()

	relvecangle := absvecangle

	// On passe de 0° / 360° à -180° / +180°
	relvecangle = trigo.FullCircleAngleToSignedHalfCircleAngle(absvecangle)

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
