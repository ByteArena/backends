package game

import (
	"encoding/json"
	"math"

	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/ecs"
)

func (deathmatch *DeathmatchGame) ProcessMutations(mutations []protocol.AgentMutationBatch) {

	for _, batch := range mutations {

		nbmutations := 0

		// Ordering actions
		// This is important because operations like shooting are taken from the previous position of the agent
		// 1. Non-movement actions (shoot, etc.)
		// 2. Movement actions

		// 1. No movement actions
		for _, mutation := range batch.Mutations {
			switch mutation.GetMethod() {
			case "shoot":
				{
					var vec []float64
					err := json.Unmarshal(mutation.GetArguments(), &vec)
					if err != nil {
						utils.Debug("arenaserver-mutation", "Failed to unmarshal JSON arguments for shoot mutation, coming from agent "+batch.AgentProxyUUID.String()+"; "+err.Error())
						continue
					}

					nbmutations++
					mutationShoot(deathmatch, batch.AgentEntityId, vector.MakeVector2(vec[0], vec[1]))

					break
				}
			}
		}

		// 2. Movement actions
		for _, mutation := range batch.Mutations {
			switch mutation.GetMethod() {
			case "steer":
				{
					var vec []float64
					err := json.Unmarshal(mutation.GetArguments(), &vec)
					if err != nil {
						utils.Debug("arenaserver-mutation", "Failed to unmarshal JSON arguments for steer mutation, coming from agent "+batch.AgentProxyUUID.String()+"; "+err.Error())
						continue
					}

					nbmutations++
					mutationSteer(deathmatch, batch.AgentEntityId, vector.MakeVector2(vec[0], vec[1]))

					break
				}
			}
		}

	}
}

func mutationSteer(game *DeathmatchGame, entityid ecs.EntityID, steering vector.Vector2) {

	tag := ecs.BuildTag(game.physicalBodyComponent)
	entityresult := game.GetEntity(entityid, tag)
	if entityresult == nil {
		return
	}

	physicalAspect := game.CastPhysicalBody(entityresult.Components[game.physicalBodyComponent.GetID()])

	velocity := physicalAspect.GetVelocity()
	orientation := physicalAspect.GetOrientation()

	prevmag := velocity.Mag()
	diff := steering.Mag() - prevmag

	maxSteeringForce := physicalAspect.GetMaxSteeringForce()
	maxAngularVelocity := physicalAspect.GetMaxAngularVelocity()
	maxSpeed := physicalAspect.GetMaxSpeed()
	if math.Abs(diff) > maxSteeringForce {
		if diff > 0 {
			steering = steering.SetMag(prevmag + maxSteeringForce)
		} else {
			steering = steering.SetMag(prevmag - maxSteeringForce)
		}
	}

	abssteering := trigo.LocalAngleToAbsoluteAngleVec(orientation, steering, &maxAngularVelocity)
	physicalAspect.SetVelocity(abssteering.Limit(maxSpeed))
}

func mutationShoot(game *DeathmatchGame, entityid ecs.EntityID, aiming vector.Vector2) {

	// //
	// // Levels consumption
	// //

	// if agentstate.LastShot <= agentstate.ShootCooldown {
	// 	// invalid shot, cooldown not over
	// 	return agentstate
	// }

	// if agentstate.ShootEnergy < agentstate.ShootEnergyCost {
	// 	// TODO(jerome): puiser dans le shield ?
	// 	return agentstate
	// }

	// agentstate.LastShot = 0
	// agentstate.ShootEnergy -= agentstate.ShootEnergyCost

	// ///////////////////////////////////////////////////////////////////////////
	// ///////////////////////////////////////////////////////////////////////////
	// // Make physical body for projectile
	// ///////////////////////////////////////////////////////////////////////////
	// ///////////////////////////////////////////////////////////////////////////

	tag := ecs.BuildTag(game.physicalBodyComponent)
	entityresult := game.GetEntity(entityid, tag)
	if entityresult == nil {
		return
	}

	entity := entityresult.Entity
	physicalAspect := game.CastPhysicalBody(entityresult.Components[game.physicalBodyComponent.GetID()])

	position := physicalAspect.GetPosition()
	orientation := physicalAspect.GetOrientation()

	// // // on passe le vecteur de visée d'un angle relatif à un angle absolu
	absaiming := trigo.LocalAngleToAbsoluteAngleVec(orientation, aiming, nil) // TODO: replace nil here by an actual angle constraint

	// FIXME(jerome): handle proper Box2D <=> BA velocity conversion
	pvel := absaiming.SetMag(100) // projectile speed; 60 is 3u/tick

	///////////////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////////

	game.NewEntityBallisticProjectile(entity.GetID(), position, pvel)
}
