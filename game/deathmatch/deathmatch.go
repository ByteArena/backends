package deathmatch

import (
	"encoding/json"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/arenaserver/types"
	commontypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/game/common"
	"github.com/bytearena/ecs"
)

type DeathmatchGame struct {
	ticknum int

	gameDescription commontypes.GameDescriptionInterface
	manager         *ecs.Manager

	physicalBodyComponent *ecs.Component
	healthComponent       *ecs.Component
	playerComponent       *ecs.Component
	renderComponent       *ecs.Component
	scriptComponent       *ecs.Component
	ttlComponent          *ecs.Component
	perceptionComponent   *ecs.Component
	ownedComponent        *ecs.Component
	steeringComponent     *ecs.Component
	shootingComponent     *ecs.Component
	impactorComponent     *ecs.Component
	collidableComponent   *ecs.Component

	agentsView     *ecs.View
	ttlView        *ecs.View
	renderableView *ecs.View
	physicalView   *ecs.View
	perceptorsView *ecs.View
	shootingView   *ecs.View
	steeringView   *ecs.View
	impactorView   *ecs.View

	PhysicalWorld     *box2d.B2World
	collisionListener *collisionListener
}

func NewDeathmatchGame(gameDescription commontypes.GameDescriptionInterface) *DeathmatchGame {
	manager := ecs.NewManager()

	game := &DeathmatchGame{
		gameDescription: gameDescription,
		manager:         manager,

		physicalBodyComponent: manager.NewComponent(),
		healthComponent:       manager.NewComponent(),
		playerComponent:       manager.NewComponent(),
		renderComponent:       manager.NewComponent(),
		scriptComponent:       manager.NewComponent(),
		ttlComponent:          manager.NewComponent(),
		perceptionComponent:   manager.NewComponent(),
		ownedComponent:        manager.NewComponent(),
		steeringComponent:     manager.NewComponent(),
		shootingComponent:     manager.NewComponent(),
		impactorComponent:     manager.NewComponent(),
		collidableComponent:   manager.NewComponent(),
	}

	gravity := box2d.MakeB2Vec2(0.0, 0.0) // gravity 0: the simulation is seen from the top
	world := box2d.MakeB2World(gravity)
	game.PhysicalWorld = &world

	initPhysicalWorld(game)

	game.ttlView = manager.CreateView(game.ttlComponent)

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
	)

	game.impactorView = manager.CreateView(
		game.impactorComponent,
		game.physicalBodyComponent,
	)

	game.physicalBodyComponent.SetDestructor(func(entity *ecs.Entity, data interface{}) {
		physicalAspect := game.CastPhysicalBody(data)
		game.PhysicalWorld.DestroyBody(physicalAspect.GetBody())
	})

	game.collisionListener = newCollisionListener(game)
	game.PhysicalWorld.SetContactListener(game.collisionListener)
	game.PhysicalWorld.SetContactFilter(newCollisionFilter(game))

	return game
}

func (deathmatch DeathmatchGame) getEntity(id ecs.EntityID, tagelements ...interface{}) *ecs.QueryResult {
	return deathmatch.manager.GetEntityByID(id, tagelements...)
}

// <GameInterface>

func (deathmatch *DeathmatchGame) ImplementsGameInterface() {}

func (deathmatch *DeathmatchGame) Subscribe(event string, cbk func(data interface{})) common.GameEventSubscription {
	return common.GameEventSubscription(0)
}

func (deathmatch *DeathmatchGame) Unsubscribe(subscription common.GameEventSubscription) {}

func (deathmatch *DeathmatchGame) Step(ticknum int, dt float64, mutations []types.AgentMutationBatch) {

	deathmatch.ticknum = ticknum

	///////////////////////////////////////////////////////////////////////////
	// On supprime les projectiles en fin de vie
	///////////////////////////////////////////////////////////////////////////
	systemTtl(deathmatch)

	///////////////////////////////////////////////////////////////////////////
	// On traite les mutations
	///////////////////////////////////////////////////////////////////////////
	systemMutations(deathmatch, mutations)

	///////////////////////////////////////////////////////////////////////////
	// On traite les tirs
	///////////////////////////////////////////////////////////////////////////
	systemShooting(deathmatch)

	///////////////////////////////////////////////////////////////////////////
	// On traite les déplacements
	///////////////////////////////////////////////////////////////////////////
	systemSteering(deathmatch)

	///////////////////////////////////////////////////////////////////////////
	// On met l'état des objets physiques à jour
	///////////////////////////////////////////////////////////////////////////
	systemPhysics(deathmatch, dt)

	///////////////////////////////////////////////////////////////////////////
	// On identifie les collisions
	///////////////////////////////////////////////////////////////////////////
	collisions := systemCollisions(deathmatch)

	///////////////////////////////////////////////////////////////////////////
	// On réagit aux collisions
	///////////////////////////////////////////////////////////////////////////
	systemHealth(deathmatch, collisions)

	///////////////////////////////////////////////////////////////////////////
	// On construit les perceptions
	///////////////////////////////////////////////////////////////////////////
	systemPerception(deathmatch)
}

func (deathmatch *DeathmatchGame) GetAgentPerception(entityid ecs.EntityID) []byte {
	entityResult := deathmatch.getEntity(entityid, deathmatch.perceptionComponent)
	if entityResult == nil {
		return []byte{}
	}

	perceptionAspect := deathmatch.CastPerception(entityResult.Components[deathmatch.perceptionComponent])
	return perceptionAspect.GetPerception()
}

func (deathmatch *DeathmatchGame) GetVizFrameJson() []byte {
	msg := commontypes.VizMessage{
		GameID:  deathmatch.gameDescription.GetId(),
		Objects: []commontypes.VizMessageObject{},
	}

	for _, entityresult := range deathmatch.renderableView.Get() {

		renderAspect := deathmatch.CastRender(entityresult.Components[deathmatch.renderComponent])
		physicalBodyAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent])

		msg.Objects = append(msg.Objects, commontypes.VizMessageObject{
			Id:          entityresult.Entity.GetID().String(),
			Type:        renderAspect.GetType(),
			Position:    physicalBodyAspect.GetPosition(),
			Velocity:    physicalBodyAspect.GetVelocity(),
			Radius:      physicalBodyAspect.GetRadius(),
			Orientation: physicalBodyAspect.GetOrientation(),
		})
	}

	res, _ := json.Marshal(msg)
	return res
}

// </GameInterface>

func initPhysicalWorld(deathmatch *DeathmatchGame) {

	arenaMap := deathmatch.gameDescription.GetMapContainer()

	// Static obstacles formed by the grounds
	for _, ground := range arenaMap.Data.Grounds {
		for _, polygon := range ground.Outline {
			deathmatch.NewEntityGround(polygon)
		}
	}

	// Explicit obstacles
	for _, obstacle := range arenaMap.Data.Obstacles {
		polygon := obstacle.Polygon
		deathmatch.NewEntityObstacle(polygon)
	}
}
