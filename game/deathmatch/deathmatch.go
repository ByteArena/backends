package deathmatch

import (
	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/arenaserver/types"
	commontypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/ecs"
	"github.com/go-gl/mathgl/mgl64"
)

type DeathmatchGame struct {
	ticknum int

	gameDescription commontypes.GameDescriptionInterface
	manager         *ecs.Manager

	physicalToAgentSpaceTransform   mgl64.Mat4
	physicalToAgentSpaceTranslation [3]float64
	physicalToAgentSpaceRotation    [3]float64
	physicalToAgentSpaceScale       float64

	physicalToAgentSpaceInverseTransform   mgl64.Mat4
	physicalToAgentSpaceInverseTranslation [3]float64
	physicalToAgentSpaceInverseRotation    [3]float64
	physicalToAgentSpaceInverseScale       float64

	physicalBodyComponent *ecs.Component
	healthComponent       *ecs.Component
	playerComponent       *ecs.Component
	renderComponent       *ecs.Component
	scriptComponent       *ecs.Component
	perceptionComponent   *ecs.Component
	ownedComponent        *ecs.Component
	steeringComponent     *ecs.Component
	shootingComponent     *ecs.Component
	impactorComponent     *ecs.Component
	collidableComponent   *ecs.Component
	lifecycleComponent    *ecs.Component
	respawnComponent      *ecs.Component

	agentsView     *ecs.View
	renderableView *ecs.View
	physicalView   *ecs.View
	perceptorsView *ecs.View
	shootingView   *ecs.View
	steeringView   *ecs.View
	impactorView   *ecs.View
	lifecycleView  *ecs.View
	respawnView    *ecs.View

	PhysicalWorld     *box2d.B2World
	collisionListener *collisionListener

	log *DeathmatchGameLog
}

func NewDeathmatchGame(gameDescription commontypes.GameDescriptionInterface) *DeathmatchGame {
	manager := ecs.NewManager()

	game := &DeathmatchGame{
		gameDescription: gameDescription,
		manager:         manager,

		physicalToAgentSpaceTransform:        mgl64.Ident4(),
		physicalToAgentSpaceInverseTransform: mgl64.Ident4(),

		physicalBodyComponent: manager.NewComponent(),
		healthComponent:       manager.NewComponent(),
		playerComponent:       manager.NewComponent(),
		renderComponent:       manager.NewComponent(),
		scriptComponent:       manager.NewComponent(),
		perceptionComponent:   manager.NewComponent(),
		ownedComponent:        manager.NewComponent(),
		steeringComponent:     manager.NewComponent(),
		shootingComponent:     manager.NewComponent(),
		impactorComponent:     manager.NewComponent(),
		collidableComponent:   manager.NewComponent(),
		lifecycleComponent:    manager.NewComponent(),
		respawnComponent:      manager.NewComponent(),

		log: NewDeathmatchGameLog(),
	}

	game.setPhysicalToAgentSpaceTransform(
		100.0,               // scale
		[3]float64{0, 0, 0}, // translation
		[3]float64{0, 0, 0}, // rotation
	)

	gravity := box2d.MakeB2Vec2(0.0, 0.0) // gravity 0: the simulation is seen from the top
	world := box2d.MakeB2World(gravity)
	game.PhysicalWorld = &world

	initPhysicalWorld(game)

	game.physicalView = manager.CreateView(game.physicalBodyComponent)

	game.perceptorsView = manager.CreateView(game.perceptionComponent)

	game.agentsView = manager.CreateView(
		game.playerComponent,
		game.physicalBodyComponent,
	)

	game.renderableView = manager.CreateView(
		game.renderComponent,
		game.physicalBodyComponent,
	)

	game.shootingView = manager.CreateView(
		game.shootingComponent,
		game.physicalBodyComponent,
	)

	game.steeringView = manager.CreateView(
		game.steeringComponent,
		game.physicalBodyComponent,
		game.lifecycleComponent,
	)

	game.impactorView = manager.CreateView(
		game.impactorComponent,
		game.physicalBodyComponent,
	)

	game.lifecycleView = manager.CreateView(
		game.lifecycleComponent,
	)

	game.respawnView = manager.CreateView(
		game.respawnComponent,
	)

	game.physicalBodyComponent.SetDestructor(func(entity *ecs.Entity, data interface{}) {
		physicalAspect := data.(*PhysicalBody)
		game.PhysicalWorld.DestroyBody(physicalAspect.GetBody())
	})

	game.collisionListener = newCollisionListener(game)
	game.PhysicalWorld.SetContactListener(game.collisionListener)
	game.PhysicalWorld.SetContactFilter(newCollisionFilter(game))

	return game
}

func (deathmatch *DeathmatchGame) setPhysicalToAgentSpaceTransform(scale float64, translation, rotation [3]float64) *DeathmatchGame {

	deathmatch.physicalToAgentSpaceScale = scale
	deathmatch.physicalToAgentSpaceTranslation = translation
	deathmatch.physicalToAgentSpaceRotation = rotation

	rotxM := mgl64.HomogRotate3DX(mgl64.DegToRad(deathmatch.physicalToAgentSpaceRotation[0]))
	rotyM := mgl64.HomogRotate3DY(mgl64.DegToRad(deathmatch.physicalToAgentSpaceRotation[1]))
	rotzM := mgl64.HomogRotate3DZ(mgl64.DegToRad(deathmatch.physicalToAgentSpaceRotation[2]))
	transM := mgl64.Translate3D(deathmatch.physicalToAgentSpaceTranslation[0], deathmatch.physicalToAgentSpaceTranslation[1], deathmatch.physicalToAgentSpaceTranslation[2])
	scaleM := mgl64.Scale3D(deathmatch.physicalToAgentSpaceScale, deathmatch.physicalToAgentSpaceScale, deathmatch.physicalToAgentSpaceScale)

	deathmatch.physicalToAgentSpaceTransform = mgl64.Ident4().
		Mul4(transM).
		Mul4(rotzM).
		Mul4(rotyM).
		Mul4(rotxM).
		Mul4(scaleM)

	deathmatch.physicalToAgentSpaceInverseScale = 1 / scale
	deathmatch.physicalToAgentSpaceInverseTranslation = [3]float64{translation[0] * -1, translation[1] * -1, translation[2] * -1}
	deathmatch.physicalToAgentSpaceInverseRotation = [3]float64{rotation[0] * -1, rotation[1] * -1, rotation[2] * -1}

	deathmatch.physicalToAgentSpaceInverseTransform = deathmatch.physicalToAgentSpaceTransform.Inv()

	return deathmatch
}

func (deathmatch DeathmatchGame) getEntity(id ecs.EntityID, tagelements ...interface{}) *ecs.QueryResult {
	return deathmatch.manager.GetEntityByID(id, tagelements...)
}

// <GameInterface>

func (deathmatch *DeathmatchGame) ImplementsGameInterface() {}

func (deathmatch *DeathmatchGame) Step(ticknum int, dt float64, mutations []types.AgentMutationBatch) {

	//watch := utils.MakeStopwatch("deathmatch::Step()")
	//watch.Start("Step")

	deathmatch.ticknum = ticknum
	respawnersTag := ecs.BuildTag(deathmatch.respawnComponent)

	///////////////////////////////////////////////////////////////////////////
	// On fait mourir les non respawners début du tour (donc après le tour
	// précédent et la construction du message de visualisation du tour précédent).
	// Cela permet de conserver la vision des projectiles à l'endroit de leur disparition pendant 1 tick
	// Pour une meilleur précision de la position de collision dans la visualisation
	///////////////////////////////////////////////////////////////////////////

	//watch.Start("systemDeath")
	systemDeath(deathmatch, respawnersTag.Inverse())
	//watch.Stop("systemDeath")

	///////////////////////////////////////////////////////////////////////////
	// On traite les mutations
	///////////////////////////////////////////////////////////////////////////
	//watch.Start("systemMutations")
	systemMutations(deathmatch, mutations)
	//watch.Stop("systemMutations")

	///////////////////////////////////////////////////////////////////////////
	// On traite les tirs
	///////////////////////////////////////////////////////////////////////////
	//watch.Start("systemShooting")
	systemShooting(deathmatch)
	//watch.Stop("systemShooting")

	///////////////////////////////////////////////////////////////////////////
	// On traite les déplacements
	///////////////////////////////////////////////////////////////////////////
	//watch.Start("systemSteering")
	systemSteering(deathmatch)
	//watch.Stop("systemSteering")

	///////////////////////////////////////////////////////////////////////////
	// On met l'état des objets physiques à jour
	///////////////////////////////////////////////////////////////////////////
	//watch.Start("systemPhysics")
	systemPhysics(deathmatch, dt)
	//watch.Stop("systemPhysics")

	///////////////////////////////////////////////////////////////////////////
	// On identifie les collisions
	///////////////////////////////////////////////////////////////////////////
	//watch.Start("systemCollisions")
	collisions := systemCollisions(deathmatch)
	//watch.Stop("systemCollisions")

	///////////////////////////////////////////////////////////////////////////
	// On réagit aux collisions
	///////////////////////////////////////////////////////////////////////////
	//watch.Start("systemHealth")
	systemHealth(deathmatch, collisions)
	//watch.Stop("systemHealth")

	///////////////////////////////////////////////////////////////////////////
	// On fait vivre les entités
	///////////////////////////////////////////////////////////////////////////
	//watch.Start("systemLifecycle")
	systemLifecycle(deathmatch)
	//watch.Stop("systemLifecycle")

	///////////////////////////////////////////////////////////////////////////
	// On fait mourir les respawners tués au cours du tour
	///////////////////////////////////////////////////////////////////////////
	//watch.Start("systemDeath")
	systemDeath(deathmatch, respawnersTag)
	//watch.Stop("systemDeath")

	///////////////////////////////////////////////////////////////////////////
	// On ressuscite les entités qui peuvent l'être
	///////////////////////////////////////////////////////////////////////////
	//watch.Start("systemRespawn")
	systemRespawn(deathmatch)
	//watch.Stop("systemRespawn")

	///////////////////////////////////////////////////////////////////////////
	// On construit les perceptions
	///////////////////////////////////////////////////////////////////////////
	//watch.Start("systemPerception")
	systemPerception(deathmatch)
	//watch.Stop("systemPerception")

	///////////////////////////////////////////////////////////////////////////
	// On supprime les entités marquées comme à supprimer
	// à la fin du tour pour éviter que box2D ne nile pas les références lors du disposeEntities
	///////////////////////////////////////////////////////////////////////////
	//watch.Start("systemDeleteEntities")
	systemDeleteEntities(deathmatch)
	//watch.Stop("systemDeleteEntities")

	//watch.Stop("Step")
	//fmt.Println(watch.String())
}

func (deathmatch *DeathmatchGame) GetAgentPerception(entityid ecs.EntityID) []byte {
	entityResult := deathmatch.getEntity(entityid, deathmatch.perceptionComponent)
	perceptionAspect := entityResult.Components[deathmatch.perceptionComponent].(*Perception)
	bytes, _ := perceptionAspect.GetPerception().MarshalJSON()
	return bytes
}

func (deathmatch *DeathmatchGame) GetAgentWelcome(entityid ecs.EntityID) []byte {

	entityresult := deathmatch.getEntity(entityid,
		deathmatch.physicalBodyComponent,
		deathmatch.steeringComponent,
		deathmatch.perceptionComponent,
	)

	if entityresult == nil {
		return []byte{}
	}

	p := agentSpecs{}

	physicalAspect := entityresult.Components[deathmatch.physicalBodyComponent].(*PhysicalBody)
	steeringAspect := entityresult.Components[deathmatch.steeringComponent].(*Steering)
	perceptionAspect := entityresult.Components[deathmatch.perceptionComponent].(*Perception)

	// TODO: this radius value comes out of box2D, and thus has to be scaled up (unlike other props)
	p.BodyRadius = physicalAspect.GetRadius() * deathmatch.physicalToAgentSpaceScale
	p.MaxSpeed = physicalAspect.GetMaxSpeed()
	p.MaxAngularVelocity = physicalAspect.GetMaxAngularVelocity()

	p.MaxSteeringForce = steeringAspect.GetMaxSteeringForce()

	p.VisionRadius = perceptionAspect.GetVisionRadius()
	p.VisionAngle = commontypes.Angle(perceptionAspect.GetVisionAngle())

	res, _ := p.MarshalJSON()
	return res
}

func (deathmatch *DeathmatchGame) GetVizFrameJson() []byte {
	msg := commontypes.VizMessage{
		GameID:        deathmatch.gameDescription.GetId(),
		Objects:       []commontypes.VizMessageObject{},
		DebugPoints:   make([][2]float64, 0),
		DebugSegments: make([][2][2]float64, 0),
	}

	for _, entityresult := range deathmatch.renderableView.Get() {

		renderAspect := entityresult.Components[deathmatch.renderComponent].(*Render)
		physicalBodyAspect := entityresult.Components[deathmatch.physicalBodyComponent].(*PhysicalBody)

		msg.Objects = append(msg.Objects, commontypes.VizMessageObject{
			Id:   entityresult.Entity.GetID().String(),
			Type: renderAspect.GetType(),

			// Here, viz coord space and physical world coord space match
			// No transform is therefore needed
			Position:    physicalBodyAspect.GetPosition(),
			Velocity:    physicalBodyAspect.GetVelocity(),
			Radius:      physicalBodyAspect.GetRadius(),
			Orientation: physicalBodyAspect.GetOrientation(),
		})

		//msg.DebugPoints = append(msg.DebugPoints, renderAspect.DebugPoints...)
		msg.DebugSegments = append(msg.DebugSegments, renderAspect.DebugSegments...)
	}

	res, _ := msg.MarshalJSON()
	return res
}

// </GameInterface>

func initPhysicalWorld(deathmatch *DeathmatchGame) {

	arenaMap := deathmatch.gameDescription.GetMapContainer()

	// Static obstacles formed by the grounds
	for _, ground := range arenaMap.Data.Grounds {
		deathmatch.NewEntityGround(ground.Polygon, ground.Name)
	}

	// Explicit obstacles
	for _, obstacle := range arenaMap.Data.Obstacles {
		polygon := obstacle.Polygon
		deathmatch.NewEntityObstacle(polygon, obstacle.Name)
	}
}
