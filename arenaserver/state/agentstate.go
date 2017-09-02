package state

import (
	"math"

	"github.com/bytearena/bytearena/arenaserver/projectile"
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
	agentId   uuid.UUID
	agentName string

	Radius             float64
	Mass               float64
	Position           vector.Vector2
	Velocity           vector.Vector2
	Orientation        float64 // heading angle in radian (degree ?) relative to arena north
	MaxSteeringForce   float64 // maximum magnitude the steering force applied to current velocity
	MaxSpeed           float64 // maximum magnitude of the agent velocity
	MaxAngularVelocity float64
	DragForce          float64 // drag opposed to the vehicle velocity at every tick turn

	Tag string // attractor

	VisionRadius float64 // radius of vision circle
	VisionAngle  float64 // angle of FOV

	MaxLife float64 // Const
	Life    float64 // Current life level; when <=0, boom

	MaxShield           float64 // Const
	Shield              float64 // Current shield level
	ShieldReplenishRate float64 // Const; Shield regained every tick

	MaxShootEnergy           float64 // Const; When shooting, energy decreases
	ShootEnergy              float64 // Current energy level
	ShootEnergyReplenishRate float64 // Const; Energy regained every tick
	ShootEnergyCost          float64 // Const; Energy consumed by a shot
	ShootCooldown            int     // Const; number of ticks to wait between every shot
	LastShot                 int     // Number of ticks since last shot

	DebugNbHits int    // Number of ticks since last shot
	DebugMsg    string // Number of ticks since last shot
}

func MakeAgentState(agentId uuid.UUID, agentName string, start mapcontainer.MapStart) AgentState {
	initialx := start.Point.X
	initialy := start.Point.Y

	r := 0.5 // agent diameter=1.0

	return AgentState{
		agentId:   agentId,
		agentName: agentName,

		Position:           vector.MakeVector2(initialx, initialy),
		Velocity:           vector.MakeNullVector2(),
		MaxSpeed:           1.50,
		MaxSteeringForce:   0.24,
		DragForce:          0.03,
		MaxAngularVelocity: number.DegreeToRadian(9), // en radians/tick; Pi = 180°
		Radius:             r,
		Mass:               math.Pi * r * r,
		Tag:                "agent",
		VisionRadius:       100,
		VisionAngle:        number.DegreeToRadian(180),

		MaxLife: 1000, // Const
		Life:    1000, // Current life level

		MaxShield:           1000, // Const
		Shield:              1000, // Current shield level
		ShieldReplenishRate: 10,   // Const; Shield regained every tick

		MaxShootEnergy:           200, // Const; When shooting, energy decreases
		ShootEnergy:              200, // Current energy level
		ShootEnergyReplenishRate: 5,   // Const; Energy regained every tick
		ShootCooldown:            5,   // Const; number of ticks to wait between every shot
		ShootEnergyCost:          0,   // Const
		LastShot:                 0,   // Number of ticks since last shot; 0 => cannot shoot immediately, must wait for first cooldown

		DebugNbHits: 0,
	}
}

func (state AgentState) GetName() string {
	return state.agentName
}

func (state AgentState) SetName(name string) AgentState {
	state.agentName = name
	return state
}

func (state AgentState) Update() AgentState {
	//newPosition := state.Position.Add(state.Velocity)
	//x, y := newPosition.Get()
	// if x < 0 || y < 0 || x > 1000 || y > 1000 {
	// 	// nothing
	// } else {
	// 	state.Position = state.Position.Add(state.Velocity)
	// }

	//
	// Apply drag to velocity
	//
	if state.DragForce > state.Velocity.Mag() {
		state.Velocity = vector.MakeNullVector2()
	} else {
		state.Velocity = state.Velocity.Sub(state.Velocity.Clone().SetMag(state.DragForce))
		state.Position = state.Position.Add(state.Velocity)
		state.Orientation = state.Velocity.Angle()
	}

	//
	// Levels replenishment
	//

	// Shield
	state.Shield += state.ShieldReplenishRate
	if state.Shield > state.MaxShield {
		state.Shield = state.MaxShield
	}

	// Energy
	state.ShootEnergy += state.ShootEnergyReplenishRate
	if state.ShootEnergy > state.MaxShootEnergy {
		state.ShootEnergy = state.MaxShootEnergy
	}

	//
	// Shoot cooldown
	//
	state.LastShot++

	return state
}

func (state AgentState) mutationSteer(steering vector.Vector2) AgentState {
	//return state

	prevmag := state.Velocity.Mag()
	diff := steering.Mag() - prevmag
	if math.Abs(diff) > state.MaxSteeringForce {
		if diff > 0 {
			steering = steering.SetMag(prevmag + state.MaxSteeringForce)
		} else {
			steering = steering.SetMag(prevmag - state.MaxSteeringForce)
		}
	}
	abssteering := localAngleToAbsoluteAngleVec(state.Orientation, steering, &state.MaxAngularVelocity)
	state.Velocity = abssteering.Limit(state.MaxSpeed)

	return state
}

func (state AgentState) mutationShoot(serverstate *ServerState, aiming vector.Vector2) AgentState {

	//
	// Levels consumption
	//

	if state.LastShot <= state.ShootCooldown {
		// invalid shot, cooldown not over
		return state
	}

	if state.ShootEnergy < state.ShootEnergyCost {
		// TODO(jerome): puiser dans le shield ?
		return state
	}

	state.LastShot = 0
	state.ShootEnergy -= state.ShootEnergyCost

	projectile := projectile.NewBallisticProjectile()
	projectile.AgentEmitterId = state.agentId
	projectile.JustFired = true

	// // on passe le vecteur de visée d'un angle relatif à un angle absolu
	absaiming := localAngleToAbsoluteAngleVec(state.Orientation, aiming, nil) // TODO: replace nil here by an actual angle constraint
	projectile.Velocity = absaiming.SetMag(projectile.Speed)                  // adding the agent position to "absolutize" the target vector

	projectile.Position = state.Position

	serverstate.SetProjectile(projectile.Id, projectile)

	return state
}

func localAngleToAbsoluteAngleVec(abscurrentagentangle float64, vec vector.Vector2, maxangleconstraint *float64) vector.Vector2 {

	// On passe de 0° / 360° à -180° / +180°
	relvecangle := trigo.FullCircleAngleToSignedHalfCircleAngle(vec.Angle())

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
