package game

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils/number"
	"github.com/bytearena/bytearena/common/utils/vector"
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
		game.PhysicalWorld.DestroyBody(physicalAspect.body)
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
func (deathmatch *DeathmatchGame) Subscribe(event string, cbk func(data interface{})) GameEventSubscription {
	return GameEventSubscription(0)
}
func (deathmatch *DeathmatchGame) Unsubscribe(subscription GameEventSubscription) {}

func (deathmatch *DeathmatchGame) Step(dt float64) {

	///////////////////////////////////////////////////////////////////////////
	// On supprime les projectiles en fin de vie
	///////////////////////////////////////////////////////////////////////////

	entitiesToRemove := make([]*ecs.Entity, 0)

	for _, entityresult := range deathmatch.ttlView.Get() {
		ttlAspect := deathmatch.CastTtl(entityresult.Components[deathmatch.ttlComponent.GetID()])
		if ttlAspect.Decrement(1) < 0 {
			entitiesToRemove = append(entitiesToRemove, entityresult.Entity)
		}
	}

	deathmatch.manager.DisposeEntities(entitiesToRemove...)

	// ///////////////////////////////////////////////////////////////////////////
	// // On met l'état des agents à jour
	// ///////////////////////////////////////////////////////////////////////////

	for _, entityresult := range deathmatch.physicalView.Get() {
		physicalAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent.GetID()])
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

			ttlAspect := deathmatch.CastTtl(entityresult.Components[deathmatch.ttlComponent.GetID()])
			physicalAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent.GetID()])

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

			ttlAspect := deathmatch.CastTtl(entityresult.Components[deathmatch.ttlComponent.GetID()])
			physicalAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent.GetID()])

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

func (deathmatch DeathmatchGame) CastScript(data interface{}) *Script {
	return data.(*Script)
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

func (deathmatch *DeathmatchGame) NewEntityAgent(position vector.Vector2) *ecs.Entity {

	agent := deathmatch.manager.NewEntity()

	bodydef := box2d.MakeB2BodyDef()
	bodydef.Position.Set(position.GetX(), position.GetY())
	bodydef.Type = box2d.B2BodyType.B2_dynamicBody
	bodydef.AllowSleep = false
	bodydef.FixedRotation = true

	body := deathmatch.PhysicalWorld.CreateBody(&bodydef)

	shape := box2d.MakeB2CircleShape()
	shape.SetRadius(0.5)

	fixturedef := box2d.MakeB2FixtureDef()
	fixturedef.Shape = &shape
	fixturedef.Density = 20.0
	body.CreateFixtureFromDef(&fixturedef)
	body.SetUserData(types.MakePhysicalBodyDescriptor(
		types.PhysicalBodyDescriptorType.Agent,
		strconv.Itoa(int(agent.GetID())),
	))
	body.SetBullet(true)
	//body.SetLinearDamping(agentstate.DragForce * float64(s.tickspersec)) // aerodynamic drag

	return agent.
		AddComponent(deathmatch.physicalBodyComponent, &PhysicalBody{
			body:               body,
			maxSpeed:           0.75,
			maxSteeringForce:   0.12,
			maxAngularVelocity: number.DegreeToRadian(9),
			dragForce:          0.015,
		}).
		AddComponent(deathmatch.perceptionComponent, &Perception{
			visionAngle:  number.DegreeToRadian(180),
			visionRadius: 100,
		}).
		AddComponent(deathmatch.healthComponent, &Health{}).
		AddComponent(deathmatch.playerComponent, &Player{}).
		AddComponent(deathmatch.renderComponent, &Render{
			type_:  "agent",
			static: false,
		}).
		AddComponent(deathmatch.scriptComponent, &Script{})
}

func (deathmatch *DeathmatchGame) NewEntityBallisticProjectile(ownerid ecs.EntityID, position vector.Vector2, velocity vector.Vector2) *ecs.Entity {

	projectile := deathmatch.manager.NewEntity()

	bodydef := box2d.MakeB2BodyDef()
	bodydef.Type = box2d.B2BodyType.B2_dynamicBody
	bodydef.AllowSleep = false
	bodydef.FixedRotation = true

	bodydef.Position.Set(position.GetX(), position.GetY())
	bodydef.LinearVelocity = box2d.MakeB2Vec2(velocity.GetX(), velocity.GetY())

	body := deathmatch.PhysicalWorld.CreateBody(&bodydef)
	body.SetLinearDamping(0.0) // no aerodynamic drag

	shape := box2d.MakeB2CircleShape()
	shape.SetRadius(0.3)

	fixturedef := box2d.MakeB2FixtureDef()
	fixturedef.Shape = &shape
	fixturedef.Density = 20.0
	body.CreateFixtureFromDef(&fixturedef)
	body.SetUserData(types.MakePhysicalBodyDescriptor(
		types.PhysicalBodyDescriptorType.Projectile,
		strconv.Itoa(int(projectile.GetID())),
	))
	body.SetBullet(true)

	return projectile.
		AddComponent(deathmatch.physicalBodyComponent, &PhysicalBody{
			body:               body,
			maxSpeed:           100,
			maxSteeringForce:   100,
			maxAngularVelocity: 10,
			dragForce:          0,
		}).
		AddComponent(deathmatch.renderComponent, &Render{
			type_:  "projectile",
			static: false,
		}).
		AddComponent(deathmatch.scriptComponent, &Script{}).
		AddComponent(deathmatch.ttlComponent, &Ttl{60}).
		AddComponent(deathmatch.ownedComponent, &Owned{ownerid})
}

// func (deathmatch *DeathmatchGame) NewEntityObstacle() *ecs.Entity {
// 	return deathmatch.manager.NewEntity().
// 		AddComponent(deathmatch.physicalBodyComponent, &PhysicalBody{}).
// 		AddComponent(deathmatch.renderComponent, &Render{static: true})
// }

///////////////////////////////////////////////////////////////////////////////
// Components structs
///////////////////////////////////////////////////////////////////////////////

type PhysicalBody struct {
	body               *box2d.B2Body
	maxSpeed           float64 // expressed in m/tick
	maxSteeringForce   float64 // expressed in m/tick
	maxAngularVelocity float64 // expressed in rad/tick
	visionRadius       float64 // expressed in m
	visionAngle        float64 // expressed in rad
	dragForce          float64 // expressed in m/tick
}

func (p *PhysicalBody) SetBody(body *box2d.B2Body) *PhysicalBody {
	p.body = body
	return p
}

func (p PhysicalBody) GetPosition() vector.Vector2 {
	v := p.body.GetPosition()
	return vector.MakeVector2(v.X, v.Y)
}

func (p *PhysicalBody) SetPosition(v vector.Vector2) *PhysicalBody {
	p.body.SetTransform(v.ToB2Vec2(), p.GetOrientation())
	return p
}

func (p PhysicalBody) GetVelocity() vector.Vector2 {
	v := p.body.GetLinearVelocity()
	return vector.MakeVector2(v.X, v.Y)
}

func (p *PhysicalBody) SetVelocity(v vector.Vector2) *PhysicalBody {
	// FIXME(jerome): properly convert units from m/tick to m/s for Box2D
	p.body.SetLinearVelocity(v.Scale(20).ToB2Vec2())
	return p
}

func (p PhysicalBody) GetOrientation() float64 {
	return p.body.GetAngle()
}

func (p *PhysicalBody) SetOrientation(angle float64) *PhysicalBody {
	// Could also be implemented using torque; see http://www.iforce2d.net/b2dtut/rotate-to-angle
	p.body.SetTransform(p.body.GetPosition(), angle)
	return p
}

func (p PhysicalBody) GetRadius() float64 {
	// FIXME(jerome): here we suppose that the agent is always a circle
	return p.body.GetFixtureList().GetShape().GetRadius()
}

func (p PhysicalBody) GetMaxSpeed() float64 {
	return p.maxSpeed
}

func (p *PhysicalBody) SetMaxSpeed(maxSpeed float64) *PhysicalBody {
	p.maxSpeed = maxSpeed
	return p
}

func (p PhysicalBody) GetMaxSteeringForce() float64 {
	return p.maxSteeringForce
}

func (p *PhysicalBody) SetMaxSteeringForce(maxSteeringForce float64) *PhysicalBody {
	p.maxSteeringForce = maxSteeringForce
	return p
}

func (p PhysicalBody) GetMaxAngularVelocity() float64 {
	return p.maxAngularVelocity
}

func (p *PhysicalBody) SetMaxAngularVelocity(maxAngularVelocity float64) *PhysicalBody {
	p.maxAngularVelocity = maxAngularVelocity
	return p
}

func (p PhysicalBody) GetVisionRadius() float64 {
	return p.visionRadius
}

func (p *PhysicalBody) SetVisionRadius(visionRadius float64) *PhysicalBody {
	p.visionRadius = visionRadius
	return p
}

func (p PhysicalBody) GetVisionAngle() float64 {
	return p.visionAngle
}

func (p *PhysicalBody) SetVisionAngle(visionAngle float64) *PhysicalBody {
	p.visionAngle = visionAngle
	return p
}

func (p PhysicalBody) GetDragForce() float64 {
	return p.dragForce
}

func (p *PhysicalBody) SetDragForce(dragForce float64) *PhysicalBody {
	p.dragForce = dragForce
	return p
}

type Health struct{}
type Player struct{}
type Render struct {
	type_  string
	static bool
}

func (r Render) GetType() string {
	return r.type_
}

type Script struct{}

type Ttl struct {
	ttl int
}

func (t *Ttl) SetValue(ttl int) *Ttl {
	t.ttl = ttl
	return t
}

func (t *Ttl) Decrement(amount int) int {
	t.ttl -= amount
	return t.ttl
}

func (t *Ttl) Increment(amount int) int {
	t.ttl += amount
	return t.ttl
}

func (t Ttl) GetValue(ttl int) int {
	return t.ttl
}

func (t *Ttl) Step() *Ttl {
	t.ttl -= 1
	return t
}

type Perception struct {
	visionAngle  float64 // expressed in rad
	visionRadius float64 // expressed in rad
}

func (p Perception) GetVisionAngle() float64 {
	return p.visionAngle
}

func (p Perception) GetVisionRadius() float64 {
	return p.visionRadius
}

type Owned struct {
	owner ecs.EntityID
}

func (o Owned) GetOwner() ecs.EntityID {
	return o.owner
}

func (o *Owned) SetOwner(owner ecs.EntityID) *Owned {
	o.owner = owner
	return o
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

		ownedAspect := filter.game.CastOwned(projectileresult.Components[filter.game.ownedComponent.GetID()])

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

		renderAspect := deathmatch.CastRender(entityresult.Components[deathmatch.renderComponent.GetID()])
		physicalBodyAspect := deathmatch.CastPhysicalBody(entityresult.Components[deathmatch.physicalBodyComponent.GetID()])

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
