package deathmatch

import (
	json "encoding/json"
	"errors"

	"github.com/bytearena/bytearena/arenaserver/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/ecs"
)

func systemMutations(deathmatch *DeathmatchGame, mutations []types.AgentMutationBatch) {

	for _, batch := range mutations {

		entityresult := deathmatch.getEntity(batch.AgentEntityId, deathmatch.lifecycleComponent)
		if entityresult != nil {
			lifecycleAspect := entityresult.Components[deathmatch.lifecycleComponent].(*Lifecycle)
			if lifecycleAspect.locked {

				// Entity is locked; discarding all mutations
				continue
			}
		}

		// Ordering actions
		// This is important because operations like shooting are taken from the previous position of the agent
		// 1. Non-movement actions (shoot, etc.)
		// 2. Movement actions

		// 1. No movement actions
		for _, mutation := range batch.Mutations {
			switch mutation.GetMethod() {
			case "shoot":
				{
					if err := handleShootMutationMessage(deathmatch, batch.AgentEntityId, mutation); err != nil {
						utils.Debug("arenaserver-mutation", err.Error()+"; coming from agent "+batch.AgentProxyUUID.String())
					}
				}
			}
		}

		// 2. Movement actions
		for _, mutation := range batch.Mutations {
			switch mutation.GetMethod() {
			case "steer":
				{
					if err := handleSteerMutationMessage(deathmatch, batch.AgentEntityId, mutation); err != nil {
						utils.Debug("arenaserver-mutation", err.Error()+"; coming from agent "+batch.AgentProxyUUID.String())
					}
				}
			}
		}

	}
}

func handleShootMutationMessage(deathmatch *DeathmatchGame, entityID ecs.EntityID, mutation types.AgentMessagePayloadActions) error {

	var aimingFloats []float64
	err := json.Unmarshal(mutation.GetArguments(), &aimingFloats)
	if err != nil {
		return errors.New("Failed to unmarshal JSON arguments for shoot mutation")
	}

	entityresult := deathmatch.getEntity(entityID, deathmatch.shootingComponent)
	if entityresult == nil {
		return errors.New("Failed to find entity associated to shoot mutation")
	}

	aiming := vector.MakeVector2(aimingFloats[0], aimingFloats[1]) //.
	//Transform(deathmatch.physicalToAgentSpaceInverseTransform)

	shootingAspect := entityresult.Components[deathmatch.shootingComponent].(*Shooting)
	shootingAspect.PushShot(aiming)

	return nil
}

func handleSteerMutationMessage(deathmatch *DeathmatchGame, entityID ecs.EntityID, mutation types.AgentMessagePayloadActions) error {
	var steeringFloats []float64
	err := json.Unmarshal(mutation.GetArguments(), &steeringFloats)
	if err != nil {
		return errors.New("Failed to unmarshal JSON arguments for steer mutation")
	}

	entityresult := deathmatch.getEntity(entityID, deathmatch.steeringComponent)
	if entityresult == nil {
		return errors.New("Failed to find entity associated to steer mutation")
	}

	steering := vector.MakeVector2(steeringFloats[0], steeringFloats[1])

	steeringAspect := entityresult.Components[deathmatch.steeringComponent].(*Steering)
	steeringAspect.PushSteer(steering)

	return nil
}
