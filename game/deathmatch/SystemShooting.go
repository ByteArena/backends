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

		position := physicalAspect.GetPosition().Transform(deathmatch.physicalToAgentSpaceTransform)
		orientation := physicalAspect.GetOrientation()

		// // // on passe le vecteur de visée d'un angle relatif à un angle absolu
		absaiming := trigo.LocalAngleToAbsoluteAngleVec(orientation, aiming, nil) // TODO: replace nil here by an actual angle constraint
		absaiming = absaiming.SetMag(30) // projectile speed; 60 is 3u/tick

		physicalSpaceAiming := absaiming.Transform(deathmatch.physicalToAgentSpaceInverseTransform)
		physicalSpacePosition := position.Transform(deathmatch.physicalToAgentSpaceInverseTransform)

		///////////////////////////////////////////////////////////////////////////
		///////////////////////////////////////////////////////////////////////////

		deathmatch.NewEntityBallisticProjectile(entity.GetID(), physicalSpacePosition, physicalSpaceAiming)
	}
}
