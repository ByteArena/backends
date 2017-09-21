package arenaserver

import (
	"log"
	"time"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/arenaserver/projectile"
	"github.com/bytearena/bytearena/common/types"
	uuid "github.com/satori/go.uuid"
)

func (server *Server) update() {

	server.debugNbUpdates++
	server.debugNbMutations++

	server.state.ProcessMutations()

	///////////////////////////////////////////////////////////////////////////
	// On supprime les projectiles en fin de vie
	///////////////////////////////////////////////////////////////////////////

	server.state.Projectilesmutex.Lock()

	projectilesToRemove := make([]uuid.UUID, 0)
	for _, projectile := range server.state.Projectiles {
		if projectile.TTL <= 0 {
			projectilesToRemove = append(projectilesToRemove, projectile.Id)
		}
	}

	server.state.ProjectilesDeletedThisTick = make(map[uuid.UUID]*projectile.BallisticProjectile)
	for _, projectileToRemoveId := range projectilesToRemove {
		// has been set to 0 during the previous tick; pruning now (0 TTL projectiles might still have a collision later in this method)

		projectile := server.state.Projectiles[projectileToRemoveId]

		// Remove projectile from moving rtree
		server.state.ProjectilesDeletedThisTick[projectileToRemoveId] = server.state.Projectiles[projectileToRemoveId]

		server.state.PhysicalWorld.DestroyBody(projectile.PhysicalBody)

		// Remove projectile from projectiles array
		delete(server.state.Projectiles, projectileToRemoveId)
	}

	///////////////////////////////////////////////////////////////////////////
	// On met l'état des projectiles à jour
	///////////////////////////////////////////////////////////////////////////

	for _, projectile := range server.state.Projectiles {
		projectile.Update()
	}

	server.state.Projectilesmutex.Unlock()

	///////////////////////////////////////////////////////////////////////////
	// On met l'état des agents à jour
	///////////////////////////////////////////////////////////////////////////

	for _, agent := range server.agents {
		id := agent.GetId()
		agentstate := server.state.GetAgentState(id)
		agentstate = agentstate.Update()
		server.state.SetAgentState(
			id,
			agentstate,
		)
	}

	///////////////////////////////////////////////////////////////////////////
	// On simule le monde physique
	///////////////////////////////////////////////////////////////////////////

	before := time.Now()

	timeStep := 1.0 / float64(server.GetTicksPerSecond())

	server.state.PhysicalWorld.Step(
		timeStep,
		8, // velocityIterations; higher improves stability; default 8 in testbed
		3, // positionIterations; higher improve overlap resolution; default 3 in testbed
	)

	log.Println("Physical world step took ", float64(time.Now().UnixNano()-before.UnixNano())/1000000.0, "ms")

	///////////////////////////////////////////////////////////////////////////
	// On réagit aux contacts
	///////////////////////////////////////////////////////////////////////////

	for _, collision := range server.collisionListener.PopCollisions() {

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
			projectileuuid, _ := uuid.FromString(descriptorCollider.ID)
			projectile := server.state.GetProjectile(projectileuuid)

			worldManifold := box2d.MakeB2WorldManifold()
			collision.GetWorldManifold(&worldManifold)

			projectile.TTL = 0
			projectile.PhysicalBody.SetLinearVelocity(box2d.MakeB2Vec2(0, 0))
			projectile.PhysicalBody.SetTransform(worldManifold.Points[0], projectile.PhysicalBody.GetAngle())

			server.state.SetProjectile(
				projectileuuid,
				projectile,
			)
		}

		if descriptorCollidee.Type == types.PhysicalBodyDescriptorType.Projectile {
			// on impacte le collider
			projectileuuid, _ := uuid.FromString(descriptorCollidee.ID)
			projectile := server.state.GetProjectile(projectileuuid)

			worldManifold := box2d.MakeB2WorldManifold()
			collision.GetWorldManifold(&worldManifold)

			projectile.TTL = 0
			projectile.PhysicalBody.SetLinearVelocity(box2d.MakeB2Vec2(0, 0))
			projectile.PhysicalBody.SetTransform(worldManifold.Points[0], projectile.PhysicalBody.GetAngle())

			server.state.SetProjectile(
				projectileuuid,
				projectile,
			)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
// Collision Handling
///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

type CollisionFilter struct { /* implements box2d.B2World.B2ContactFilterInterface */
	server *Server
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
		projectileid, _ := uuid.FromString(projectile.ID)
		p := filter.server.GetState().GetProjectile(projectileid)
		return p.AgentEmitterId.String() != other.ID
	}

	return true
}

func newCollisionFilter(server *Server) *CollisionFilter {
	return &CollisionFilter{
		server: server,
	}
}

type CollisionListener struct { /* implements box2d.B2World.B2ContactListenerInterface */
	server          *Server
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

func newCollisionListener(server *Server) *CollisionListener {
	return &CollisionListener{
		server: server,
	}
}

///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
