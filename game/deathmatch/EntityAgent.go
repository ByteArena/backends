package deathmatch

import (
	"log"
	"math/rand"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/number"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/ecs"
)

func (deathmatch *DeathmatchGame) NewEntityAgent(spawnPosition vector.Vector2) *ecs.Entity {

	agent := deathmatch.manager.NewEntity()

	///////////////////////////////////////////////////////////////////////////
	// Définition de ses caractéristiques physiques de l'agent (spécifications)
	///////////////////////////////////////////////////////////////////////////

	// Linear unit expressed in agent space units (meters) per tick
	// Angular unit expressed in radians per tick

	bodyRadius := 0.5
	maxSpeed := 1.25
	maxSteering := 10000.0
	dragForce := 0.015
	maxAngularVelocity := number.DegreeToRadian(15.0)

	visionRadius := 150.0
	visionAngle := number.DegreeToRadian(160)

	///////////////////////////////////////////////////////////////////////////
	// Création du corps physique de l'agent (Box2D)
	///////////////////////////////////////////////////////////////////////////

	bodydef := box2d.MakeB2BodyDef()
	bodydef.Position.Set(spawnPosition.GetX(), spawnPosition.GetY())
	bodydef.Type = box2d.B2BodyType.B2_dynamicBody
	bodydef.AllowSleep = false
	bodydef.FixedRotation = true

	body := deathmatch.PhysicalWorld.CreateBody(&bodydef)

	shape := box2d.MakeB2CircleShape()
	shape.SetRadius(bodyRadius * deathmatch.physicalToAgentSpaceInverseScale)

	fixturedef := box2d.MakeB2FixtureDef()
	fixturedef.Shape = &shape
	fixturedef.Density = 20.0
	body.CreateFixtureFromDef(&fixturedef)
	body.SetUserData(types.MakePhysicalBodyDescriptor(
		types.PhysicalBodyDescriptorType.Agent,
		agent.GetID(),
	))
	body.SetBullet(false)

	///////////////////////////////////////////////////////////////////////////
	// Composition de l'agent dans l'ECS
	///////////////////////////////////////////////////////////////////////////

	tps := deathmatch.gameDescription.GetTps()

	return agent.
		AddComponent(deathmatch.physicalBodyComponent, &PhysicalBody{
			body:               body,
			maxSpeed:           maxSpeed,
			maxAngularVelocity: maxAngularVelocity,
			dragForce:          dragForce,

			pointTransformIn:  deathmatch.physicalToAgentSpaceInverseTransform,
			pointTransformOut: deathmatch.physicalToAgentSpaceTransform,

			distanceScaleIn:  deathmatch.physicalToAgentSpaceInverseScale, // same as transform matrix, but scale only (for 1D transforms of length)
			distanceScaleOut: deathmatch.physicalToAgentSpaceScale,        // same as transform matrix, but scale only (for 1D transforms of length)

			timeScaleIn:  float64(tps),       // m/tick to m/s; => ticksPerSecond
			timeScaleOut: 1.0 / float64(tps), // m/s to m/tick; => 1 / ticksPerSecond
		}).
		AddComponent(deathmatch.perceptionComponent, &Perception{
			visionAngle:  visionAngle,
			visionRadius: visionRadius,
		}).
		AddComponent(deathmatch.healthComponent, NewHealth(100)).
		AddComponent(deathmatch.playerComponent, &Player{}).
		AddComponent(deathmatch.renderComponent, &Render{
			type_:       "agent",
			static:      false,
			DebugPoints: make([][2]float64, 0),
		}).
		AddComponent(deathmatch.shootingComponent, NewShooting()).
		AddComponent(deathmatch.steeringComponent, NewSteering(
			maxSteering, // MaxSteering
		)).
		AddComponent(deathmatch.collidableComponent, NewCollidable(
			CollisionGroup.Agent,
			utils.BuildTag(
				CollisionGroup.Agent,
				CollisionGroup.Obstacle,
				CollisionGroup.Projectile,
				CollisionGroup.Ground,
			),
		).SetCollisionScriptFunc(agentCollisionScript)).
		AddComponent(deathmatch.lifecycleComponent, &Lifecycle{
			onDeath: func() {
				log.Println("AGENT DEATH !!!!!!!!!!!!!!!!!!!!!")
				qr := deathmatch.getEntity(agent.GetID(), deathmatch.respawnComponent, deathmatch.lifecycleComponent)
				if qr == nil {
					// should never happen
					return
				}

				respawnAspect := qr.Components[deathmatch.respawnComponent].(*Respawn)
				lifecycleAspect := qr.Components[deathmatch.lifecycleComponent].(*Lifecycle)

				lifecycleAspect.locked = true

				respawnAspect.isRespawning = true
				respawnAspect.respawningCountdown = deathmatch.gameDescription.GetTps() * 5 // 5 seconds

			},
		}).
		AddComponent(deathmatch.respawnComponent, &Respawn{
			onRespawn: func() {

				qr := deathmatch.getEntity(agent.GetID(),
					deathmatch.physicalBodyComponent,
					deathmatch.lifecycleComponent,
					deathmatch.healthComponent,
				)

				if qr == nil {
					// should never happen
					return
				}

				// pick a randown spawn point
				starts := deathmatch.gameDescription.GetMapContainer().Data.Starts
				spawnPoint := starts[rand.Int()%(len(starts)-1)].Point

				physicalAspect := qr.Components[deathmatch.physicalBodyComponent].(*PhysicalBody)
				lifecycleAspect := qr.Components[deathmatch.lifecycleComponent].(*Lifecycle)
				healthAspect := qr.Components[deathmatch.healthComponent].(*Health)

				physicalAspect.SetPosition(vector.MakeVector2(spawnPoint.GetX(), spawnPoint.GetY()))
				lifecycleAspect.locked = false
				healthAspect.Restore()
			},
		})
}

func agentCollisionScript(game *DeathmatchGame, entityID ecs.EntityID, otherEntityID ecs.EntityID, collidableAspect *Collidable, otherCollidableAspectB *Collidable, point vector.Vector2) {
	entityResult := game.getEntity(entityID, game.physicalBodyComponent)
	if entityResult == nil {
		return
	}

	physicalAspect := entityResult.Components[game.physicalBodyComponent].(*PhysicalBody)
	physicalAspect.SetVelocity(vector.MakeNullVector2())
}
