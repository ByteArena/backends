package deathmatch

import (
	"encoding/json"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/arenaserver/types"
	commontypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/game/common"
	"github.com/bytearena/ecs"
)

type DeathmatchGame struct {
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

	agentsView     *ecs.View
	ttlView        *ecs.View
	renderableView *ecs.View
	physicalView   *ecs.View
	perceptorsView *ecs.View

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

		PhysicalWorld: buildPhysicalWorld(gameDescription.GetMapContainer()),
	}

	game.agentsView = manager.CreateView("agents", ecs.BuildTag(
		game.playerComponent, game.physicalBodyComponent,
	))

	game.ttlView = manager.CreateView("ttlbound", ecs.BuildTag(
		game.ttlComponent,
	))

	game.renderableView = manager.CreateView("renderable", ecs.BuildTag(
		game.renderComponent, game.physicalBodyComponent,
	))

	game.physicalView = manager.CreateView("physicalbodies", ecs.BuildTag(
		game.physicalBodyComponent,
	))

	game.perceptorsView = manager.CreateView("perceptors", ecs.BuildTag(
		game.perceptionComponent,
	))

	game.physicalBodyComponent.SetDestructor(func(entity *ecs.Entity, data interface{}) {
		physicalAspect := game.CastPhysicalBody(data)
		game.PhysicalWorld.DestroyBody(physicalAspect.GetBody())
	})

	game.collisionListener = newCollisionListener(game)
	game.PhysicalWorld.SetContactListener(game.collisionListener)
	game.PhysicalWorld.SetContactFilter(newCollisionFilter(game))

	return game
}

func (deathmatch DeathmatchGame) getEntity(id ecs.EntityID, tag ecs.Tag) *ecs.QueryResult {
	return deathmatch.manager.GetEntityByID(id, tag)
}

// <GameInterface>

func (deathmatch *DeathmatchGame) ImplementsGameInterface() {}

func (deathmatch *DeathmatchGame) Subscribe(event string, cbk func(data interface{})) common.GameEventSubscription {
	return common.GameEventSubscription(0)
}

func (deathmatch *DeathmatchGame) Unsubscribe(subscription common.GameEventSubscription) {}

func (deathmatch *DeathmatchGame) Step(dt float64, mutations []types.AgentMutationBatch) {

	///////////////////////////////////////////////////////////////////////////
	// On supprime les projectiles en fin de vie
	///////////////////////////////////////////////////////////////////////////
	systemTtl(deathmatch)

	///////////////////////////////////////////////////////////////////////////
	// On traite les mutations
	///////////////////////////////////////////////////////////////////////////
	systemMutations(deathmatch, mutations)

	///////////////////////////////////////////////////////////////////////////
	// On met l'état des objets physiques à jour
	///////////////////////////////////////////////////////////////////////////
	systemPhysics(deathmatch, dt)

	///////////////////////////////////////////////////////////////////////////
	// On réagit aux contacts
	///////////////////////////////////////////////////////////////////////////
	systemCollisions(deathmatch)

	///////////////////////////////////////////////////////////////////////////
	// On construit les perceptions
	///////////////////////////////////////////////////////////////////////////
	systemPerception(deathmatch)
}

func (deathmatch *DeathmatchGame) GetAgentPerception(entityid ecs.EntityID) []byte {
	tag := ecs.BuildTag(deathmatch.perceptionComponent)
	entityResult := deathmatch.getEntity(entityid, tag)
	if entityResult == nil {
		return []byte{}
	}

	perceptionAspect := deathmatch.CastPerception(entityResult.Components[deathmatch.perceptionComponent])
	return perceptionAspect.GetPerception()
}

func (deathmatch *DeathmatchGame) ProduceVizMessageJson() []byte {
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

func buildPhysicalWorld(arenaMap *mapcontainer.MapContainer) *box2d.B2World {

	// Define the gravity vector.
	gravity := box2d.MakeB2Vec2(0.0, 0.0) // 0: the simulation is seen from the top

	// Construct a world object, which will hold and simulate the rigid bodies.
	world := box2d.MakeB2World(gravity)

	// Static obstacles formed by the grounds
	for _, ground := range arenaMap.Data.Grounds {
		for _, polygon := range ground.Outline {

			bodydef := box2d.MakeB2BodyDef()
			bodydef.Type = box2d.B2BodyType.B2_staticBody

			body := world.CreateBody(&bodydef)
			vertices := make([]box2d.B2Vec2, len(polygon.Points)-1) // -1: avoid last point because the last point of the loop should not be repeated

			for i := 0; i < len(polygon.Points)-1; i++ {
				vertices[i].Set(polygon.Points[i].X, polygon.Points[i].Y)
			}

			shape := box2d.MakeB2ChainShape()
			shape.CreateLoop(vertices, len(vertices))
			body.CreateFixture(&shape, 0.0)
			body.SetUserData(commontypes.MakePhysicalBodyDescriptor(commontypes.PhysicalBodyDescriptorType.Ground, ground.Id))
		}
	}

	// Explicit obstacles
	for _, obstacle := range arenaMap.Data.Obstacles {
		polygon := obstacle.Polygon
		bodydef := box2d.MakeB2BodyDef()
		bodydef.Type = box2d.B2BodyType.B2_staticBody

		body := world.CreateBody(&bodydef)
		vertices := make([]box2d.B2Vec2, len(polygon.Points)-1) // a polygon has as many edges as points

		for i := 0; i < len(polygon.Points)-1; i++ {
			vertices[i].Set(polygon.Points[i].X, polygon.Points[i].Y)
		}

		shape := box2d.MakeB2ChainShape()
		shape.CreateLoop(vertices, len(vertices))
		body.CreateFixture(&shape, 0.0)
		body.SetUserData(commontypes.MakePhysicalBodyDescriptor(commontypes.PhysicalBodyDescriptorType.Obstacle, obstacle.Id))
	}

	return &world
}
