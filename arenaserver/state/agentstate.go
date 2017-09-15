package state

import (
	"math"

	b2collision "github.com/bytearena/box2d/box2d/collision"
	b2common "github.com/bytearena/box2d/box2d/common"
	b2dynamics "github.com/bytearena/box2d/box2d/dynamics"

	"github.com/bytearena/bytearena/arenaserver/projectile"
	"github.com/bytearena/bytearena/common/types"
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

	PhysicalBody *b2dynamics.B2Body // replaces Radius, Mass, Position, Velocity, Orientation

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

func MakeAgentState(agentId uuid.UUID, agentName string, physicalbody *b2dynamics.B2Body) AgentState {

	return AgentState{
		agentId:   agentId,
		agentName: agentName,

		PhysicalBody: physicalbody,

		MaxSpeed:           9,
		MaxSteeringForce:   3,
		DragForce:          0.015,
		MaxAngularVelocity: number.DegreeToRadian(9), // en radians/tick; Pi = 180°

		Tag:          "agent",
		VisionRadius: 100,
		VisionAngle:  number.DegreeToRadian(180),

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

	if state.GetVelocity().Mag() > 0.01 {
		state.SetOrientation(state.GetVelocity().Angle())
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

	prevmag := state.GetVelocity().Mag()
	diff := steering.Mag() - prevmag
	if math.Abs(diff) > state.MaxSteeringForce {
		if diff > 0 {
			steering = steering.SetMag(prevmag + state.MaxSteeringForce)
		} else {
			steering = steering.SetMag(prevmag - state.MaxSteeringForce)
		}
	}
	abssteering := localAngleToAbsoluteAngleVec(state.GetOrientation(), steering, &state.MaxAngularVelocity)
	state.SetVelocity(abssteering.Limit(state.MaxSpeed))

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

	projectileId := uuid.NewV4()

	///////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////////
	// Make physical body for projectile
	///////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////////

	agentpos := state.GetPosition()

	bodydef := b2dynamics.MakeB2BodyDef()
	bodydef.Type = b2dynamics.B2BodyType.B2_dynamicBody
	bodydef.AllowSleep = false
	bodydef.FixedRotation = true

	bodydef.Position.Set(agentpos.GetX(), agentpos.GetY())

	// // on passe le vecteur de visée d'un angle relatif à un angle absolu
	absaiming := localAngleToAbsoluteAngleVec(state.GetOrientation(), aiming, nil) // TODO: replace nil here by an actual angle constraint
	pvel := absaiming.SetMag(20)                                                   // projectile speed
	bodydef.LinearVelocity = b2common.MakeB2Vec2(pvel.GetX(), pvel.GetY())

	body := serverstate.PhysicalWorld.CreateBody(&bodydef)

	shape := b2collision.MakeB2CircleShape()
	shape.SetRadius(0.3)

	fixturedef := b2dynamics.MakeB2FixtureDef()
	fixturedef.Shape = &shape
	fixturedef.Density = 20.0
	body.CreateFixture(&fixturedef)
	body.SetUserData(types.MakePhysicalBodyDescriptor(types.PhysicalBodyDescriptorType.Projectile, projectileId.String()))

	///////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////////

	projectile := projectile.NewBallisticProjectile(projectileId, body)
	projectile.AgentEmitterId = state.agentId
	projectile.JustFired = true
	projectile.TTL = 60

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

func (state *AgentState) SetVelocity(velocity vector.Vector2) {
	state.PhysicalBody.SetLinearVelocity(
		b2common.MakeB2Vec2(velocity.GetX(), velocity.GetY()),
	)
}

func (state AgentState) GetVelocity() vector.Vector2 {
	v := state.PhysicalBody.GetLinearVelocity()
	return vector.MakeVector2(v.X, v.Y)
}

func (state *AgentState) SetPosition(position vector.Vector2) {
	b2p := b2common.MakeB2Vec2(position.GetX(), position.GetY())
	state.PhysicalBody.SetTransform(b2p, state.PhysicalBody.GetAngle())
}

func (state AgentState) GetPosition() vector.Vector2 {
	v := state.PhysicalBody.GetPosition()
	return vector.MakeVector2(v.X, v.Y)
}

func (state *AgentState) SetOrientation(angle float64) {
	// Could also be implemented using torque; see http://www.iforce2d.net/b2dtut/rotate-to-angle
	state.PhysicalBody.SetTransform(state.PhysicalBody.GetPosition(), angle)
}

func (state AgentState) GetOrientation() float64 {
	return state.PhysicalBody.GetAngle()
}

func (state AgentState) GetRadius() float64 {
	// FIXME(jerome): here we suppose that the agent is always a circle
	return state.PhysicalBody.GetFixtureList().GetShape().GetRadius()
}
