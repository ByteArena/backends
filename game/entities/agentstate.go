package entities

// import (
// 	"github.com/bytearena/box2d"

// 	uuid "github.com/satori/go.uuid"
// )

// // Agent is a Simple Vehicle Model from Reynolds (http://www.red3d.com/cwr/steer/gdc99/)
// // Agent Physics is based on forward Euler integration
// //
// // At each simulation step, behaviorally determined steering forces (as limited by max_force)
// // are applied to the vehicle’s point mass. This produces an acceleration equal to the steering force
// // divided by the vehicle’s mass. That acceleration is added to the old velocity to produce
// // a new velocity, which is then truncated by max_speed.
// // Finally, the velocity is added to the old position:
// //
// // steering_force = limitmagnitude(steering_direction, max_force)
// // acceleration = steering_force / mass
// // velocity = limitmagnitude(velocity + acceleration, max_speed)
// // position = position + velocity
// //
// // Because of its assumption of velocity alignment, this simple vehicle model cannot simulate
// // effects such as skids, spins or slides. Furthermore this model allows the vehicle to turn
// // when its speed is zero. Most real vehicles cannot do this (they are “non-holonomic”)
// // and in any case it allows undesirably large changes in orientation during a single time step.
// //
// // This problem can be solved by placing an additional constraint on change of orientation,
// // or by limiting the lateral steering component at low speeds,
// // or by simulating moment of inertia.

// type AgentState struct {
// 	MaxLife float64 // Const
// 	Life    float64 // Current life level; when <=0, boom

// 	MaxShield           float64 // Const
// 	Shield              float64 // Current shield level
// 	ShieldReplenishRate float64 // Const; Shield regained every tick

// 	MaxShootEnergy           float64 // Const; When shooting, energy decreases
// 	ShootEnergy              float64 // Current energy level
// 	ShootEnergyReplenishRate float64 // Const; Energy regained every tick
// 	ShootEnergyCost          float64 // Const; Energy consumed by a shot
// 	ShootCooldown            int     // Const; number of ticks to wait between every shot
// 	LastShot                 int     // Number of ticks since last shot

// 	DebugNbHits int    // Number of ticks since last shot
// 	DebugMsg    string // Number of ticks since last shot
// }

// func MakeAgentState(agentproxyUUID uuid.UUID, agentName string, physicalbody *box2d.B2Body) AgentState {

// 	return AgentState{

// 		MaxLife: 1000, // Const
// 		Life:    1000, // Current life level

// 		MaxShield:           1000, // Const
// 		Shield:              1000, // Current shield level
// 		ShieldReplenishRate: 10,   // Const; Shield regained every tick

// 		MaxShootEnergy:           200, // Const; When shooting, energy decreases
// 		ShootEnergy:              200, // Current energy level
// 		ShootEnergyReplenishRate: 5,   // Const; Energy regained every tick
// 		ShootCooldown:            2,   // Const; number of ticks to wait between every shot
// 		ShootEnergyCost:          0,   // Const
// 		LastShot:                 0,   // Number of ticks since last shot; 0 => cannot shoot immediately, must wait for first cooldown

// 		DebugNbHits: 0,
// 	}
// }

// // func (state AgentState) Update() AgentState {

// // 	if state.GetVelocity().Mag() > 0.01 {
// // 		state.SetOrientation(state.GetVelocity().Angle())
// // 	}

// // 	//
// // 	// Levels replenishment
// // 	//

// // 	// Shield
// // 	state.Shield += state.ShieldReplenishRate
// // 	if state.Shield > state.MaxShield {
// // 		state.Shield = state.MaxShield
// // 	}

// // 	// Energy
// // 	state.ShootEnergy += state.ShootEnergyReplenishRate
// // 	if state.ShootEnergy > state.MaxShootEnergy {
// // 		state.ShootEnergy = state.MaxShootEnergy
// // 	}

// // 	//
// // 	// Shoot cooldown
// // 	//
// // 	state.LastShot++

// // 	return state
// // }
