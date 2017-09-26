package deathmatch

import (
	"encoding/json"

	"github.com/bytearena/bytearena/arenaserver/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/vector"
)

func systemMutations(deathmatch *DeathmatchGame, mutations []types.AgentMutationBatch) {

	for _, batch := range mutations {

		// Ordering actions
		// This is important because operations like shooting are taken from the previous position of the agent
		// 1. Non-movement actions (shoot, etc.)
		// 2. Movement actions

		// 1. No movement actions
		for _, mutation := range batch.Mutations {
			switch mutation.GetMethod() {
			case "shoot":
				{

					var aimingFloats []float64
					err := json.Unmarshal(mutation.GetArguments(), &aimingFloats)
					if err != nil {
						utils.Debug("arenaserver-mutation", "Failed to unmarshal JSON arguments for shoot mutation, coming from agent "+batch.AgentProxyUUID.String()+"; "+err.Error())
						continue
					}

					aiming := vector.MakeVector2(aimingFloats[0], aimingFloats[1])

					entityresult := deathmatch.getEntity(batch.AgentEntityId, deathmatch.shootingComponent)
					if entityresult == nil {
						continue
					}

					shootingAspect := deathmatch.CastShooting(entityresult.Components[deathmatch.shootingComponent])
					shootingAspect.PushShot(aiming)

					break
				}
			}
		}

		// 2. Movement actions
		for _, mutation := range batch.Mutations {
			switch mutation.GetMethod() {
			case "steer":
				{
					var steeringFloats []float64
					err := json.Unmarshal(mutation.GetArguments(), &steeringFloats)
					if err != nil {
						utils.Debug("arenaserver-mutation", "Failed to unmarshal JSON arguments for steer mutation, coming from agent "+batch.AgentProxyUUID.String()+"; "+err.Error())
						continue
					}

					steering := vector.MakeVector2(steeringFloats[0], steeringFloats[1])

					entityresult := deathmatch.getEntity(batch.AgentEntityId, deathmatch.steeringComponent)
					if entityresult == nil {
						continue
					}

					steeringAspect := deathmatch.CastSteering(entityresult.Components[deathmatch.steeringComponent])
					steeringAspect.PushSteer(steering)

					break
				}
			}
		}

	}
}
