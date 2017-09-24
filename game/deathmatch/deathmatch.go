package deathmatch

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/bytearena/game/common"
	"github.com/bytearena/ecs"
)

type DeathmatchGame struct {
	gameDescription types.GameDescriptionInterface
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

	PhysicalWorld     *box2d.B2World
	collisionListener *CollisionListener
}

func NewDeathmatchGame(gameDescription types.GameDescriptionInterface) *DeathmatchGame {
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

	game.physicalBodyComponent.SetDestructor(func(entity *ecs.Entity, data interface{}) {
		physicalAspect := game.CastPhysicalBody(data)
		game.PhysicalWorld.DestroyBody(physicalAspect.GetBody())
	})

	game.collisionListener = newCollisionListener(game)
	game.PhysicalWorld.SetContactListener(game.collisionListener)
	game.PhysicalWorld.SetContactFilter(newCollisionFilter(game))

	return game
}

func (deathmatch DeathmatchGame) GetEntity(id ecs.EntityID, tag ecs.Tag) *ecs.QueryResult {
	return deathmatch.manager.GetEntityByID(id, tag)
}

// <GameInterface>

func (deathmatch *DeathmatchGame) ImplementsGameInterface() {}
func (deathmatch *DeathmatchGame) Subscribe(event string, cbk func(data interface{})) common.GameEventSubscription {
	return common.GameEventSubscription(0)
}
func (deathmatch *DeathmatchGame) Unsubscribe(subscription common.GameEventSubscription) {}

func (deathmatch *DeathmatchGame) Step(dt float64) {

	///////////////////////////////////////////////////////////////////////////
	// On supprime les projectiles en fin de vie
	///////////////////////////////////////////////////////////////////////////

	entitiesToRemove := make([]*ecs.Entity, 0)

	for _, entityresult := range deathmatch.ttlView.Get() {
		ttlAspect := deathmatch.CastTtl(entityresult.Components[deathmatch.ttlComponent])
		if ttlAspect.Decrement(1) < 0 {
			entitiesToRemove = append(entitiesToRemove, entityresult.Entity)
		}
	}

	deathmatch.manager.DisposeEntities(entitiesToRemove...)

	// ///////////////////////////////////////////////////////////////////////////
	// // On met l'état des agents à jour
	// ///////////////////////////////////////////////////////////////////////////

	for _, entityresult := range deathmatch.physicalView.Get() {
		physicalAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent])
		if physicalAspect.GetVelocity().Mag() > 0.01 {
			physicalAspect.SetOrientation(physicalAspect.GetVelocity().Angle())
		}
	}

	// ///////////////////////////////////////////////////////////////////////////
	// // On simule le monde physique
	// ///////////////////////////////////////////////////////////////////////////

	before := time.Now()

	deathmatch.PhysicalWorld.Step(
		dt,
		8, // velocityIterations; higher improves stability; default 8 in testbed
		3, // positionIterations; higher improve overlap resolution; default 3 in testbed
	)

	log.Println("Physical world step took ", float64(time.Now().UnixNano()-before.UnixNano())/1000000.0, "ms")

	// ///////////////////////////////////////////////////////////////////////////
	// // On réagit aux contacts
	// ///////////////////////////////////////////////////////////////////////////

	for _, collision := range deathmatch.collisionListener.PopCollisions() {

		descriptorCollider, ok := collision.GetFixtureA().GetBody().GetUserData().(types.PhysicalBodyDescriptor)
		if !ok {
			continue
		}

		descriptorCollidee, ok := collision.GetFixtureB().GetBody().GetUserData().(types.PhysicalBodyDescriptor)
		if !ok {
			continue
		}

		if descriptorCollider.Type == types.PhysicalBodyDescriptorType.Projectile {
			// on impacte le collider
			id, _ := strconv.Atoi(descriptorCollider.ID)
			entityid := ecs.EntityID(id)
			entityresult := deathmatch.GetEntity(entityid, ecs.BuildTag(
				deathmatch.ttlComponent,
				deathmatch.playerComponent,
			))
			if entityresult == nil {
				continue
			}

			worldManifold := box2d.MakeB2WorldManifold()
			collision.GetWorldManifold(&worldManifold)

			ttlAspect := deathmatch.CastTtl(entityresult.Components[deathmatch.ttlComponent])
			physicalAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent])

			ttlAspect.SetValue(1)

			physicalAspect.
				SetVelocity(vector.MakeNullVector2()).
				SetPosition(vector.FromB2Vec2(worldManifold.Points[0]))
		}

		if descriptorCollidee.Type == types.PhysicalBodyDescriptorType.Projectile {
			// on impacte le collider
			id, _ := strconv.Atoi(descriptorCollidee.ID)
			entityid := ecs.EntityID(id)
			entityresult := deathmatch.GetEntity(entityid, ecs.BuildTag(
				deathmatch.ttlComponent,
				deathmatch.playerComponent,
			))
			if entityresult == nil {
				continue
			}

			worldManifold := box2d.MakeB2WorldManifold()
			collision.GetWorldManifold(&worldManifold)

			ttlAspect := deathmatch.CastTtl(entityresult.Components[deathmatch.ttlComponent])
			physicalAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent])

			ttlAspect.SetValue(1)

			physicalAspect.
				SetVelocity(vector.MakeNullVector2()).
				SetPosition(vector.FromB2Vec2(worldManifold.Points[0]))
		}
	}
}

// </GameInterface>

func (deathmatch DeathmatchGame) CastPhysicalBody(data interface{}) *PhysicalBody {
	return data.(*PhysicalBody)
}

func (deathmatch DeathmatchGame) GetHealth(data interface{}) *Health {
	return data.(*Health)
}

func (deathmatch DeathmatchGame) CastPlayer(data interface{}) *Player {
	return data.(*Player)
}

func (deathmatch DeathmatchGame) CastRender(data interface{}) *Render {
	return data.(*Render)
}

func (deathmatch DeathmatchGame) CastTtl(data interface{}) *Ttl {
	return data.(*Ttl)
}

func (deathmatch DeathmatchGame) CastPerception(data interface{}) *Perception {
	return data.(*Perception)
}

func (deathmatch DeathmatchGame) CastOwned(data interface{}) *Owned {
	return data.(*Owned)
}

///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
// Collision Handling
///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

type CollisionFilter struct { /* implements box2d.B2World.B2ContactFilterInterface */
	game *DeathmatchGame
}

func (filter *CollisionFilter) ShouldCollide(fixtureA *box2d.B2Fixture, fixtureB *box2d.B2Fixture) bool {
	// Si projectile, ne pas collisionner agent émetteur
	// Si projectile, ne pas collisionner ground

	descriptorA, ok := fixtureA.GetBody().GetUserData().(types.PhysicalBodyDescriptor)
	if !ok {
		return false
	}

	descriptorB, ok := fixtureB.GetBody().GetUserData().(types.PhysicalBodyDescriptor)
	if !ok {
		return false
	}

	aIsProjectile := descriptorA.Type == types.PhysicalBodyDescriptorType.Projectile
	bIsProjectile := descriptorB.Type == types.PhysicalBodyDescriptorType.Projectile

	if !aIsProjectile && !bIsProjectile {
		return true
	}

	if aIsProjectile && bIsProjectile {
		return true
	}

	var projectile *types.PhysicalBodyDescriptor
	var other *types.PhysicalBodyDescriptor

	if aIsProjectile {
		projectile = &descriptorA
		other = &descriptorB
	} else {
		projectile = &descriptorB
		other = &descriptorA
	}

	if other.Type == types.PhysicalBodyDescriptorType.Obstacle {
		return true
	}

	if other.Type == types.PhysicalBodyDescriptorType.Ground {
		return false
	}

	if other.Type == types.PhysicalBodyDescriptorType.Agent {
		// fetch projectile
		projectileid, _ := strconv.Atoi(projectile.ID)

		tag := ecs.BuildTag(filter.game.ownedComponent)
		projectileresult := filter.game.GetEntity(ecs.EntityID(projectileid), tag)
		if projectileresult == nil {
			return false
		}

		ownedAspect := filter.game.CastOwned(projectileresult.Components[filter.game.ownedComponent])

		return ownedAspect.GetOwner().String() != other.ID
	}

	return true
}

func newCollisionFilter(game *DeathmatchGame) *CollisionFilter {
	return &CollisionFilter{
		game: game,
	}
}

type CollisionListener struct { /* implements box2d.B2World.B2ContactListenerInterface */
	game            *DeathmatchGame
	collisionbuffer []box2d.B2ContactInterface
}

func (listener *CollisionListener) PopCollisions() []box2d.B2ContactInterface {
	defer func() { listener.collisionbuffer = make([]box2d.B2ContactInterface, 0) }()
	return listener.collisionbuffer
}

/// Called when two fixtures begin to touch.
func (listener *CollisionListener) BeginContact(contact box2d.B2ContactInterface) { // contact has to be backed by a pointer
	listener.collisionbuffer = append(listener.collisionbuffer, contact)
}

/// Called when two fixtures cease to touch.
func (listener *CollisionListener) EndContact(contact box2d.B2ContactInterface) { // contact has to be backed by a pointer
	//log.Println("END:COLLISION !!!!!!!!!!!!!!")
}

/// This is called after a contact is updated. This allows you to inspect a
/// contact before it goes to the solver. If you are careful, you can modify the
/// contact manifold (e.g. disable contact).
/// A copy of the old manifold is provided so that you can detect changes.
/// Note: this is called only for awake bodies.
/// Note: this is called even when the number of contact points is zero.
/// Note: this is not called for sensors.
/// Note: if you set the number of contact points to zero, you will not
/// get an EndContact callback. However, you may get a BeginContact callback
/// the next step.
func (listener *CollisionListener) PreSolve(contact box2d.B2ContactInterface, oldManifold box2d.B2Manifold) { // contact has to be backed by a pointer
	//log.Println("PRESOLVE !!!!!!!!!!!!!!")
}

/// This lets you inspect a contact after the solver is finished. This is useful
/// for inspecting impulses.
/// Note: the contact manifold does not include time of impact impulses, which can be
/// arbitrarily large if the sub-step is small. Hence the impulse is provided explicitly
/// in a separate data structure.
/// Note: this is only called for contacts that are touching, solid, and awake.
func (listener *CollisionListener) PostSolve(contact box2d.B2ContactInterface, impulse *box2d.B2ContactImpulse) { // contact has to be backed by a pointer
	//log.Println("POSTSOLVE !!!!!!!!!!!!!!")
}

func newCollisionListener(game *DeathmatchGame) *CollisionListener {
	return &CollisionListener{
		game: game,
	}
}

///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

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
			body.SetUserData(types.MakePhysicalBodyDescriptor(types.PhysicalBodyDescriptorType.Ground, ground.Id))
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
		body.SetUserData(types.MakePhysicalBodyDescriptor(types.PhysicalBodyDescriptorType.Obstacle, obstacle.Id))
	}

	return &world
}

///////////////////////////////////////////////////////////////////////////////

func (deathmatch *DeathmatchGame) ProduceVizMessageJson() []byte {
	msg := types.VizMessage{
		GameID:  deathmatch.gameDescription.GetId(),
		Objects: []types.VizMessageObject{},
	}

	for _, entityresult := range deathmatch.renderableView.Get() {

		renderAspect := deathmatch.CastRender(entityresult.Components[deathmatch.renderComponent])
		physicalBodyAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent])

		msg.Objects = append(msg.Objects, types.VizMessageObject{
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
