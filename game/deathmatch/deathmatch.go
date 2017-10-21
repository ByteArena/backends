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
		perceptionComponent:   manager.NewComponent(),
		ownedComponent:        manager.NewComponent(),
		steeringComponent:     manager.NewComponent(),
		shootingComponent:     manager.NewComponent(),
		impactorComponent:     manager.NewComponent(),
		collidableComponent:   manager.NewComponent(),
		lifecycleComponent:    manager.NewComponent(),
		respawnComponent:      manager.NewComponent(),
	}

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

	respawnersTag := ecs.BuildTag(deathmatch.respawnComponent)

	///////////////////////////////////////////////////////////////////////////
	// On fait mourir les non respawners début du tour (donc après le tour
	// précédent et la construction du message de visualisation du tour précédent).
	// Cela permet de conserver la vision des projectiles à l'endroit de leur disparition pendant 1 tick
	// Pour une meilleur précision de la position de collision dans la visualisation
	///////////////////////////////////////////////////////////////////////////

	systemDeath(deathmatch, respawnersTag.Inverse())

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
	// On fait vivre les entités
	///////////////////////////////////////////////////////////////////////////
	systemLifecycle(deathmatch)

	///////////////////////////////////////////////////////////////////////////
	// On fait mourir les respawners tués au cours du tour
	///////////////////////////////////////////////////////////////////////////
	systemDeath(deathmatch, respawnersTag)

	///////////////////////////////////////////////////////////////////////////
	// On ressuscite les entités qui peuvent l'être
	///////////////////////////////////////////////////////////////////////////
	systemRespawn(deathmatch)

	///////////////////////////////////////////////////////////////////////////
	// On construit les perceptions
	///////////////////////////////////////////////////////////////////////////
	systemPerception(deathmatch)

	///////////////////////////////////////////////////////////////////////////
	// On supprime les entités marquées comme à supprimer
	// à la fin du tour pour éviter que box2D ne nile pas les références lors du disposeEntities
	///////////////////////////////////////////////////////////////////////////
	systemDeleteEntities(deathmatch)

}

func (deathmatch *DeathmatchGame) GetAgentPerception(entityid ecs.EntityID) []byte {
	entityResult := deathmatch.getEntity(entityid, deathmatch.perceptionComponent)
	perceptionAspect := entityResult.Components[deathmatch.perceptionComponent].(*Perception)
	return perceptionAspect.GetPerception()
}

func (deathmatch *DeathmatchGame) GetVizFrameJson() []byte {
	msg := commontypes.VizMessage{
		GameID:      deathmatch.gameDescription.GetId(),
		Objects:     []commontypes.VizMessageObject{},
		DebugPoints: make([][2]float64, 0),
	}

	for _, entityresult := range deathmatch.renderableView.Get() {

		renderAspect := entityresult.Components[deathmatch.renderComponent].(*Render)
		physicalBodyAspect := entityresult.Components[deathmatch.physicalBodyComponent].(*PhysicalBody)

		msg.Objects = append(msg.Objects, commontypes.VizMessageObject{
			Id:          entityresult.Entity.GetID().String(),
			Type:        renderAspect.GetType(),
			Position:    physicalBodyAspect.GetPosition(),
			Velocity:    physicalBodyAspect.GetVelocity(),
			Radius:      physicalBodyAspect.GetRadius(),
			Orientation: physicalBodyAspect.GetOrientation(),
		})

		msg.DebugPoints = append(msg.DebugPoints, renderAspect.DebugPoints...)
	}

	res, _ := json.Marshal(msg)
	return res
}

// </GameInterface>

func initPhysicalWorld(deathmatch *DeathmatchGame) {

	arenaMap := deathmatch.gameDescription.GetMapContainer()

	// Static obstacles formed by the grounds
	for _, ground := range arenaMap.Data.Grounds {
		deathmatch.NewEntityGround(ground.Polygon)
	}

	// Explicit obstacles
	for _, obstacle := range arenaMap.Data.Obstacles {
		polygon := obstacle.Polygon
		deathmatch.NewEntityObstacle(polygon)
	}
}
