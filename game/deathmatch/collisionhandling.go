package deathmatch

import (
	"strconv"

	"github.com/bytearena/box2d"
	commontypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/ecs"
)

///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
// Collision Handling
///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

type collisionFilter struct { /* implements box2d.B2World.B2ContactFilterInterface */
	game *DeathmatchGame
}

func (filter *collisionFilter) ShouldCollide(fixtureA *box2d.B2Fixture, fixtureB *box2d.B2Fixture) bool {
	// Si projectile, ne pas collisionner agent émetteur
	// Si projectile, ne pas collisionner ground

	descriptorA, ok := fixtureA.GetBody().GetUserData().(commontypes.PhysicalBodyDescriptor)
	if !ok {
		return false
	}

	descriptorB, ok := fixtureB.GetBody().GetUserData().(commontypes.PhysicalBodyDescriptor)
	if !ok {
		return false
	}

	aIsProjectile := descriptorA.Type == commontypes.PhysicalBodyDescriptorType.Projectile
	bIsProjectile := descriptorB.Type == commontypes.PhysicalBodyDescriptorType.Projectile

	if !aIsProjectile && !bIsProjectile {
		return true
	}

	if aIsProjectile && bIsProjectile {
		return true
	}

	var projectile *commontypes.PhysicalBodyDescriptor
	var other *commontypes.PhysicalBodyDescriptor

	if aIsProjectile {
		projectile = &descriptorA
		other = &descriptorB
	} else {
		projectile = &descriptorB
		other = &descriptorA
	}

	if other.Type == commontypes.PhysicalBodyDescriptorType.Obstacle {
		return true
	}

	if other.Type == commontypes.PhysicalBodyDescriptorType.Ground {
		return false
	}

	if other.Type == commontypes.PhysicalBodyDescriptorType.Agent {
		// fetch projectile
		projectileid, _ := strconv.Atoi(projectile.ID)

		tag := ecs.BuildTag(filter.game.ownedComponent)
		projectileresult := filter.game.getEntity(ecs.EntityID(projectileid), tag)
		if projectileresult == nil {
			return false
		}

		ownedAspect := filter.game.CastOwned(projectileresult.Components[filter.game.ownedComponent])

		return ownedAspect.GetOwner().String() != other.ID
	}

	return true
}

func newCollisionFilter(game *DeathmatchGame) *collisionFilter {
	return &collisionFilter{
		game: game,
	}
}

type collisionListener struct { /* implements box2d.B2World.B2ContactListenerInterface */
	game            *DeathmatchGame
	collisionbuffer []box2d.B2ContactInterface
}

func (listener *collisionListener) PopCollisions() []box2d.B2ContactInterface {
	defer func() { listener.collisionbuffer = make([]box2d.B2ContactInterface, 0) }()
	return listener.collisionbuffer
}

/// Called when two fixtures begin to touch.
func (listener *collisionListener) BeginContact(contact box2d.B2ContactInterface) { // contact has to be backed by a pointer
	listener.collisionbuffer = append(listener.collisionbuffer, contact)
}

/// Called when two fixtures cease to touch.
func (listener *collisionListener) EndContact(contact box2d.B2ContactInterface) { // contact has to be backed by a pointer
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
func (listener *collisionListener) PreSolve(contact box2d.B2ContactInterface, oldManifold box2d.B2Manifold) { // contact has to be backed by a pointer
	//log.Println("PRESOLVE !!!!!!!!!!!!!!")
}

/// This lets you inspect a contact after the solver is finished. This is useful
/// for inspecting impulses.
/// Note: the contact manifold does not include time of impact impulses, which can be
/// arbitrarily large if the sub-step is small. Hence the impulse is provided explicitly
/// in a separate data structure.
/// Note: this is only called for contacts that are touching, solid, and awake.
func (listener *collisionListener) PostSolve(contact box2d.B2ContactInterface, impulse *box2d.B2ContactImpulse) { // contact has to be backed by a pointer
	//log.Println("POSTSOLVE !!!!!!!!!!!!!!")
}

func newCollisionListener(game *DeathmatchGame) *collisionListener {
	return &collisionListener{
		game: game,
	}
}

///////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
