package deathmatch

import (
	"github.com/bytearena/bytearena/common/utils/trigo"
)

func systemShooting(deathmatch *DeathmatchGame) {

	for _, entityresult := range deathmatch.shootingView.Get() {

		shootingAspect := entityresult.Components[deathmatch.shootingComponent].(*Shooting)
		physicalAspect := entityresult.Components[deathmatch.physicalBodyComponent].(*PhysicalBody)

		shots := shootingAspect.PopPendingShots()
		if len(shots) == 0 {
			continue
		}

		aiming := shots[0]
		entity := entityresult.Entity

		// //
		// // Levels consumption
		// //

		if deathmatch.ticknum-shootingAspect.LastShot <= shootingAspect.ShootCooldown {
			// invalid shot, cooldown not over
			continue
		}

		if shootingAspect.ShootEnergy < shootingAspect.ShootEnergyCost {
			// TODO(jerome): puiser dans le shield ?
			continue
		}

		shootingAspect.LastShot = deathmatch.ticknum
		shootingAspect.ShootEnergy -= shootingAspect.ShootEnergyCost

		// ///////////////////////////////////////////////////////////////////////////
		// ///////////////////////////////////////////////////////////////////////////
		// // Make physical body for projectile
		// ///////////////////////////////////////////////////////////////////////////
		// ///////////////////////////////////////////////////////////////////////////

		orientation := physicalAspect.GetOrientation()

		// // // on passe le vecteur de visée d'un angle relatif à un angle absolu
		velocity := trigo.
			LocalAngleToAbsoluteAngleVec(orientation, aiming, nil). // TODO: replace nil here by an actual angle constraint
			SetMag(200)

		physicalSpaceVelocity := velocity.Transform(deathmatch.physicalToAgentSpaceInverseTransform)

		// position := physicalAspect.GetPosition().Transform(deathmatch.physicalToAgentSpaceTransform)
		// physicalSpacePosition := position.Transform(deathmatch.physicalToAgentSpaceInverseTransform)
		physicalSpacePosition := physicalAspect.GetPosition()

		///////////////////////////////////////////////////////////////////////////
		///////////////////////////////////////////////////////////////////////////

		//physicalSpacePosition = vector.MakeVector2(-1.5, 0)

		deathmatch.NewEntityBallisticProjectile(entity.GetID(), physicalSpacePosition, physicalSpaceVelocity)
	}
}
