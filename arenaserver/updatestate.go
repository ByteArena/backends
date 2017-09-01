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

func handleCollisions(server *Server, beforeStateAgents map[uuid.UUID]collision.CollisionMovingObjectState, beforeStateProjectiles map[uuid.UUID]collision.CollisionMovingObjectState) {

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
			beforeStateAgents,
			server.GetState().MapMemoization,
			state.GeometryObjectType.Agent,
			nil,
			func(objectid uuid.UUID) collision.CollisionMovingObjectState {
				object := server.GetState().GetAgentState(objectid)
				return collision.CollisionMovingObjectState{
					Position: object.Position,
					Velocity: object.Velocity,
					Radius:   object.Radius,
				}
			},
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
			beforeStateProjectiles,
			server.GetState().MapMemoization,
			state.GeometryObjectType.Projectile,
			[]int{state.GeometryObjectType.ObstacleGround},
			func(objectid uuid.UUID) collision.CollisionMovingObjectState {
				object := server.GetState().GetProjectile(objectid)
				return collision.CollisionMovingObjectState{
					Position: object.Position,
					Velocity: object.Velocity,
					Radius:   object.Radius,
				}
			},
		)

		collisionsMutex.Lock()
		collisions = append(collisions, colls...)
		collisionsMutex.Unlock()
		wait.Done()
	}()

	go func() {

		movements := make([]*collision.MovementState, 0)

		///////////////////////////////////////////////////////////////////////////
		// Moving / Moving collisions
		///////////////////////////////////////////////////////////////////////////

		// Indexing agents trajectories in rtree
		for id, beforeState := range beforeStateAgents {

			agentstate := server.state.GetAgentState(id)

			afterState := collision.CollisionMovingObjectState{
				Position: agentstate.Position,
				Velocity: agentstate.Velocity,
				Radius:   agentstate.Radius,
			}

			bbRegion, err := collision.GetTrajectoryBoundingBox(
				beforeState.Position, beforeState.Radius,
				afterState.Position, afterState.Radius,
			)
			if err != nil {
				utils.Debug("arena-server-updatestate", "Error in processMovingObjectsCollisions: could not define bbRegion in moving rTree")
				return
			}

			//show.Dump(bbRegion)

			movements = append(movements, &collision.MovementState{
				Type:   state.GeometryObjectType.Agent,
				ID:     id.String(),
				Before: beforeState,
				After:  afterState,
				Rect:   bbRegion,
			})
		}

		// Indexing projectiles trajectories in rtree
		for id, beforeState := range beforeStateProjectiles {

			projectile := server.GetState().GetProjectile(id)

			afterState := collision.CollisionMovingObjectState{
				Position: projectile.Position,
				Velocity: projectile.Velocity,
				Radius:   projectile.Radius,
			}

			bbRegion, err := collision.GetTrajectoryBoundingBox(
				beforeState.Position, beforeState.Radius,
				afterState.Position, afterState.Radius,
			)
			if err != nil {
				utils.Debug("arena-server-updatestate", "Error in processMovingObjectsCollisions: could not define bbRegion in moving rTree")
				return
			}

			movements = append(movements, &collision.MovementState{
				Type:   state.GeometryObjectType.Projectile,
				ID:     id.String(),
				Before: beforeState,
				After:  afterState,
				Rect:   bbRegion,
			})
		}

		//rtMoving := server.state.MapMemoization.RtreeMoving
		spatials := make([]rtreego.Spatial, len(movements))
		for i, m := range movements {
			spatials[i] = rtreego.Spatial(m)
		}
		rtMoving := rtreego.NewTree(2, 25, 50, spatials...) // TODO(jerome): better constants here ? what heuristic to use ?

		colls := collision.ProcessMovingMovingCollisions(movements, rtMoving)
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

				if projectile.AgentEmitterId.String() == coll.CollideeID {
					continue
				}

				//log.Println("PROJECTILE TOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOUCHED")

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
				if coll.CollideeType == state.GeometryObjectType.Projectile {
					projectileuuid, _ := uuid.FromString(coll.CollideeID)
					projectile := server.state.GetProjectile(projectileuuid)
					if projectile.AgentEmitterId.String() == agentuuid.String() {
						continue
					}
				}

				//log.Println("AGENT TOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOUCHED")

				agentstate := server.GetState().GetAgentState(agentuuid)
				agentstate.Position = coll.Point
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
