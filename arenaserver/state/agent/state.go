package agent

import (
	"encoding/json"
	"math"
	"math/rand"

	serverprotocol "github.com/bytearena/bytearena/arenaserver/protocol"
	stateprotocol "github.com/bytearena/bytearena/arenaserver/state/protocol"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/number"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
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

	// Common to all kinds of agents
	Tag string // attractor

	Radius       float64
	Mass         float64
	Position     vector.Vector2
	Velocity     vector.Vector2
	Orientation  float64 // heading angle in radians relative to arena north
	VisionRadius float64 // radius of vision circle
	VisionAngle  float64 // angle of FOV

	// Holonomic drive agents
	MaxSteeringForce   float64 // maximum magnitude the steering force applied to current velocity
	MaxSpeed           float64 // maximum magnitude of the agent velocity
	MaxAngularVelocity float64
}

func MakeAgentState() AgentState {
	initialx := 100 + rand.Float64()*800
	initialy := 100 + rand.Float64()*300

	r := 6 + rand.Float64()*6.0

	return AgentState{
		Position:           vector.MakeVector2(initialx, initialy),
		Velocity:           vector.MakeVector2(0.00001, 1),
		MaxSpeed:           20.0 / 3,
		MaxSteeringForce:   1.0,
		MaxAngularVelocity: number.DegreeToRadian(6), // en radians/tick; Pi = 180°
		Radius:             r,
		Mass:               math.Pi * r * r,
		Tag:                "agent",
		VisionRadius:       400,
		VisionAngle:        number.DegreeToRadian(120),
	}
}

func (state AgentState) Update() stateprotocol.AgentStateInterface {
	newPosition := state.Position.Add(state.Velocity)
	x, y := newPosition.Get()
	if x < 0 || y < 0 || x > 1000 || y > 1000 {
		// nothing
	} else {
		state.Position = state.Position.Add(state.Velocity)
	}

	state.Orientation = state.Velocity.Angle()
	return state
}

func (state AgentState) MutationSteer(steering vector.Vector2) stateprotocol.AgentStateInterface {

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

/*
func (state AgentState) mutationShoot(serverstate *srvstate.ServerState, aiming vector.Vector2) AgentState {

	// on passe le vecteur de visée d'un angle relatif à un angle absolu
	absaiming := state.localAngleToAbsoluteAngleVec(aiming, nil)

	projectile := statepkg.ProjectileState{
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
*/

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

func (state AgentState) Clone() stateprotocol.AgentStateInterface {
	return state // yes, passed by value !
}

func (state AgentState) ProcessMutations(mutations []serverprotocol.MessageMutation) stateprotocol.AgentStateInterface {

	newstate := state.Clone()

	nbmutations := 0
	for _, mutation := range mutations {

		switch mutation.GetMethod() {
		case "steer":
			{
				var vec []float64
				err := json.Unmarshal(mutation.GetArguments(), &vec)
				utils.Check(err, "Failed to unmarshal JSON arguments for steer mutation")

				nbmutations++
				newstate = newstate.MutationSteer(vector.MakeVector2(vec[0], vec[1]))

				break
			}
			/*case "shoot":
			{
				var vec []float64
				err := json.Unmarshal(mutation.GetArguments(), &vec)
				utils.Check(err, "Failed to unmarshal JSON arguments for shoot mutation")

				nbmutations++
				newstate = newstate.mutationShoot(serverstate, vector.MakeVector2(vec[0], vec[1]))

				break
			}*/
		}
	}

	return newstate

}

func (state AgentState) GetOrientation() float64 {
	return state.Orientation
}

func (state AgentState) GetVelocity() vector.Vector2 {
	return state.Velocity
}

func (state AgentState) GetPosition() vector.Vector2 {
	return state.Position
}

func (state AgentState) GetRadius() float64 {
	return state.Radius
}

func (state AgentState) GetVisionRadius() float64 {
	return state.VisionRadius
}

func (state AgentState) GetVisionAngle() float64 {
	return state.VisionAngle
}
