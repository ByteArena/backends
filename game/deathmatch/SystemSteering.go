package deathmatch

import (
	"math"

	"github.com/bytearena/bytearena/common/utils/trigo"
)

func systemSteering(deathmatch *DeathmatchGame) {
	for _, entityresult := range deathmatch.steeringView.Get() {
		steeringAspect := deathmatch.CastSteering(entityresult.Components[deathmatch.steeringComponent])
		physicalAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent])

		steers := steeringAspect.PopPendingSteers()
		if len(steers) == 0 {
			continue
		}

		steering := steers[0]

		velocity := physicalAspect.GetVelocity()
		orientation := physicalAspect.GetOrientation()

		prevmag := velocity.Mag()
		diff := steering.Mag() - prevmag

		maxSteeringForce := steeringAspect.GetMaxSteeringForce()
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
}
