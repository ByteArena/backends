package arenaserver

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/bytearena/bytearena/arenaserver/collision"
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
	uuid "github.com/satori/go.uuid"
)

func handleCollisions(server *Server, agentMovements []*collision.MovementState, projectileMovements []*collision.MovementState) {

	// TODO(jerome): check for collisions:
	// * agent / agent
	// * agent / obstacle
	// * agent / projectile
	// * projectile / projectile
	// * projectile / obstacle

	begin := time.Now()
	//show := spew.ConfigState{MaxDepth: 5, Indent: "    "}

	collisions := make([]collision.Collision, 0)
	collisionsMutex := &sync.Mutex{}

	wait := &sync.WaitGroup{}
	wait.Add(3)

	go func() {

		///////////////////////////////////////////////////////////////////////////
		// Agents / static collisions
		///////////////////////////////////////////////////////////////////////////

		colls := collision.ProcessMovingStaticCollisions(
			agentMovements,
			server.GetState().MapMemoization,
			nil,
		)
		collisionsMutex.Lock()
		collisions = append(collisions, colls...)
		collisionsMutex.Unlock()
		wait.Done()
	}()

	go func() {

		///////////////////////////////////////////////////////////////////////////
		// Projectiles / static collisions
		///////////////////////////////////////////////////////////////////////////

		colls := collision.ProcessMovingStaticCollisions(
			projectileMovements,
			server.GetState().MapMemoization,
			[]int{state.GeometryObjectType.ObstacleGround},
		)

		collisionsMutex.Lock()
		collisions = append(collisions, colls...)
		collisionsMutex.Unlock()
		wait.Done()
	}()

	go func() {

		///////////////////////////////////////////////////////////////////////////
		// Moving / Moving collisions
		///////////////////////////////////////////////////////////////////////////

		allMovements := make([]*collision.MovementState, 0)
		allMovements = append(allMovements, agentMovements...)
		allMovements = append(allMovements, projectileMovements...)

		//rtMoving := server.state.MapMemoization.RtreeMoving
		spatials := make([]rtreego.Spatial, len(allMovements))
		for i, m := range allMovements {
			spatials[i] = rtreego.Spatial(m)
		}
		rtMoving := rtreego.NewTree(2, 25, 50, spatials...) // TODO(jerome): better constants here ? what heuristic to use ?

		colls := collision.ProcessMovingMovingCollisions(allMovements, rtMoving)
		//show.Dump("RECEIVED HERE", colls)
		collisionsMutex.Lock()
		collisions = append(collisions, colls...)
		collisionsMutex.Unlock()
		wait.Done()
	}()

	wait.Wait()

	utils.Debug("collision-detection", fmt.Sprintf("Took %f ms; found %d collisions", time.Now().Sub(begin).Seconds()*1000, len(collisions)))

	// Ordering collisions along time (causality order)
	sort.Sort(collision.CollisionByTimeAsc(collisions))

	//show.Dump("RESOLVED sorted", collisions)

	collisionsThatHappened := make([]collision.Collision, 0)
	hasAlreadyCollided := make(map[string]struct{})

	for _, coll := range collisions {
		hashkey := strconv.Itoa(coll.ColliderType) + ":" + coll.ColliderID

		if coll.ColliderType == state.GeometryObjectType.Projectile {
			projuuid, _ := uuid.FromString(coll.ColliderID)
			proj := server.state.GetProjectile(projuuid)

			if proj.AgentEmitterId.String() == coll.CollideeID {
				// Projectile cannot shoot emitter agent (happens when the projectile is right out of the agent cannon)
				continue
			}
		}

		if _, ok := hasAlreadyCollided[hashkey]; ok {
			// owner has already collided (or been collided) by another object earlier in the tick
			// this collision cannot happen (that is, if we trust causality)
			//log.Println("CAUSALITY BITCH !")
			continue
		} else {
			hasAlreadyCollided[hashkey] = struct{}{}
			collisionsThatHappened = append(collisionsThatHappened, coll)
		}
	}

	//show.Dump(collisionsThatHappened)
	//log.Println("EFFECTIVE COLLISIONS", len(collisionsThatHappened))

	for _, coll := range collisionsThatHappened {
		switch coll.ColliderType {
		case state.GeometryObjectType.Projectile:
			{
				projectileuuid, _ := uuid.FromString(coll.ColliderID)
				projectile := server.state.GetProjectile(projectileuuid)

				//log.Println("PROJECTILE TOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOUCHED")

				// projectile.Position = collision.EnsureValidPositionAfterCollision(
				// 	server.GetState().MapMemoization,
				// 	coll,
				// )
				projectile.Position = coll.Point
				projectile.Velocity = vector.MakeNullVector2()
				projectile.TTL = 0

				// if coll.otherType == state.GeometryObjectType.Projectile {
				// 	utils.Debug("collision-detection", "BOOOOOOOOOOOOOOOOOOOOOOOOOOOOOM PROJECTILES")
				// }

				server.state.SetProjectile(
					projectileuuid,
					projectile,
				)
			}
		case state.GeometryObjectType.Agent:
			{
				agentuuid, _ := uuid.FromString(coll.ColliderID)

				agentstate := server.GetState().GetAgentState(agentuuid)
				agentstate.Position = collision.EnsureValidPositionAfterCollision(
					server.GetState().MapMemoization,
					coll,
				)
				//agentstate.Orientation++
				agentstate.Velocity = vector.MakeNullVector2()

				server.state.SetAgentState(
					agentuuid,
					agentstate,
				)
			}
		}
	}
}
