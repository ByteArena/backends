package arenaserver

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
	uuid "github.com/satori/go.uuid"
)

type collision struct {
	ownerType int
	ownerID   string
	otherType int
	otherID   string
	point     vector.Vector2
	timeBegin float64 // from 0 to 1, 0 = beginning of tick, 1 = end of tick
	timeEnd   float64
}

type CollisionByTimeAsc []collision

func (a CollisionByTimeAsc) Len() int           { return len(a) }
func (a CollisionByTimeAsc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a CollisionByTimeAsc) Less(i, j int) bool { return a[i].timeBegin < a[j].timeBegin }

func handleCollisions(server *Server, beforeStateAgents map[uuid.UUID]movingObjectTemporaryState, beforeStateProjectiles map[uuid.UUID]movingObjectTemporaryState) {

	// TODO(jerome): check for collisions:
	// * agent / agent
	// * agent / obstacle
	// * agent / projectile
	// * projectile / projectile
	// * projectile / obstacle

	//show := spew.ConfigState{MaxDepth: 5, Indent: "\t"}

	begin := time.Now()

	collisions := make([]collision, 0)
	collisionsMutex := &sync.Mutex{}

	wait := &sync.WaitGroup{}
	wait.Add(3)

	go func() {

		///////////////////////////////////////////////////////////////////////////
		// Agents / static collisions
		///////////////////////////////////////////////////////////////////////////

		colls := processAgentObstacleCollisions(server, beforeStateAgents)
		collisionsMutex.Lock()
		collisions = append(collisions, colls...)
		collisionsMutex.Unlock()
		wait.Done()
	}()

	go func() {

		///////////////////////////////////////////////////////////////////////////
		// Projectiles / static collisions
		///////////////////////////////////////////////////////////////////////////

		colls := processProjectileObstacleCollisions(server, beforeStateProjectiles)
		collisionsMutex.Lock()
		collisions = append(collisions, colls...)
		collisionsMutex.Unlock()
		wait.Done()
	}()

	go func() {

		movements := make([]*movementState, 0)

		///////////////////////////////////////////////////////////////////////////
		// Moving objects collisions
		///////////////////////////////////////////////////////////////////////////

		// Indexing agents trajectories in rtree
		// TODO: index BB for displacement polygon, not displacement center line
		for id, beforeState := range beforeStateAgents {

			agentstate := server.state.GetAgentState(id)

			afterState := movingObjectTemporaryState{
				Position: agentstate.Position,
				Velocity: agentstate.Velocity,
				Radius:   agentstate.Radius,
			}

			bbRegion, err := getTrajectoryBoundingBox(
				beforeState.Position, beforeState.Radius,
				afterState.Position, afterState.Radius,
			)
			if err != nil {
				utils.Debug("arena-server-updatestate", "Error in processMovingObjectsCollisions: could not define bbRegion in moving rTree")
				return
			}

			movements = append(movements, &movementState{
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

			afterState := movingObjectTemporaryState{
				Position: projectile.Position,
				Velocity: projectile.Velocity,
				Radius:   projectile.Radius,
			}

			bbRegion, err := getTrajectoryBoundingBox(
				beforeState.Position, beforeState.Radius,
				afterState.Position, afterState.Radius,
			)
			if err != nil {
				utils.Debug("arena-server-updatestate", "Error in processMovingObjectsCollisions: could not define bbRegion in moving rTree")
				return
			}

			projectile.PreviousBounds = projectile.Bounds
			projectile.Bounds = bbRegion
			server.GetState().SetProjectile(projectile.Id, projectile)

			movements = append(movements, &movementState{
				Type:   state.GeometryObjectType.Projectile,
				ID:     id.String(),
				Before: beforeState,
				After:  afterState,
				Rect:   bbRegion,
			})
		}

		rtMoving := server.state.MapMemoization.RtreeMoving
		log.Println("PROJECTILE SIZE", rtMoving.Size())

		if rtMoving.Size() == 0 {
			// Not initialized
			for _, movement := range movements {
				rtMoving.Insert(movement)
			}
		} else {

			for _, movement := range movements {
				if !movement.After.Position.Equals(movement.Before.Position) {
					//log.Println("UPDATE AVANT", rtMoving.Size())
					projectileuuid, _ := uuid.FromString(movement.ID)
					projectile := server.GetState().GetProjectile(projectileuuid)
					if projectile != nil {
						rtMoving.DeleteWithComparator(&movementState{
							ID:   movement.ID,
							Type: movement.Type,
							Rect: projectile.PreviousBounds,
						}, movementStateComparator)
					} else {
						rtMoving.DeleteWithComparator(movement, movementStateComparator)
					}
					rtMoving.Insert(movement)
					//log.Println("UPDATE APRES", rtMoving.Size())
				} else {
					//log.Println("laaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
				}
			}

			// remove deleted projectiles
			for projectileid, projectile := range server.state.ProjectilesDeletedThisTick {
				//log.Println("AVANT", rtMoving.Size())
				rtMoving.DeleteWithComparator(&movementState{
					ID:   projectileid.String(),
					Type: state.GeometryObjectType.Projectile,
					Rect: projectile.PreviousBounds,
				}, movementStateComparator)
				//log.Println("APRES", rtMoving.Size())
			}
		}

		colls := processMovingObjectsCollisions(server, movements)
		collisionsMutex.Lock()
		collisions = append(collisions, colls...)
		collisionsMutex.Unlock()
		wait.Done()
	}()

	wait.Wait()

	utils.Debug("collision-detection", fmt.Sprintf("Took %f ms; found %d collisions", time.Now().Sub(begin).Seconds()*1000, len(collisions)))

	// Ordering collisions along time (causality order)
	sort.Sort(CollisionByTimeAsc(collisions))

	collisionsPerOwner := make(map[string]([]*collision))
	collisionsPerOther := make(map[string]([]*collision))

	for _, coll := range collisions {
		hashkeyOwner := strconv.Itoa(coll.ownerType) + ":" + coll.ownerID
		if _, ok := collisionsPerOwner[hashkeyOwner]; !ok {
			collisionsPerOwner[hashkeyOwner] = make([]*collision, 0)
		}

		hashkeyOther := strconv.Itoa(coll.otherType) + ":" + coll.otherID
		if _, ok := collisionsPerOther[hashkeyOther]; !ok {
			collisionsPerOther[hashkeyOther] = make([]*collision, 0)
		}

		collisionsPerOwner[hashkeyOwner] = append(collisionsPerOwner[hashkeyOwner], &coll)
		collisionsPerOther[hashkeyOther] = append(collisionsPerOther[hashkeyOther], &coll)
	}

	// show.Dump("collisionsPerOwner", collisionsPerOwner)
	// show.Dump("collisionsPerOther", collisionsPerOther)

	collisionsThatHappened := make([]collision, 0)
	hasAlreadyCollided := make(map[string]struct{})

	for _, coll := range collisions {
		hashkey := strconv.Itoa(coll.ownerType) + ":" + coll.ownerID

		if _, ok := hasAlreadyCollided[hashkey]; ok {
			// owner has already collided (or been collided) by another object earlier in the tick
			// this collision may not happen (that is, if we trust causality)
			//log.Println("CAUSALITY BITCH !")
			continue
		} else {
			hasAlreadyCollided[hashkey] = struct{}{}
			collisionsThatHappened = append(collisionsThatHappened, coll)
		}
	}

	//show.Dump(collisionsThatHappened)

	for _, coll := range collisionsThatHappened {
		switch coll.ownerType {
		case state.GeometryObjectType.Projectile:
			{
				projectileuuid, _ := uuid.FromString(coll.ownerID)

				projectile := server.state.GetProjectile(projectileuuid)
				projectile.Position = coll.point
				projectile.Velocity = vector.MakeNullVector2()

				server.state.SetProjectile(
					projectileuuid,
					projectile,
				)
			}
		case state.GeometryObjectType.Agent:
			{
				agentuuid, _ := uuid.FromString(coll.ownerID)

				agentstate := server.GetState().GetAgentState(agentuuid)
				agentstate.Position = coll.point
				agentstate.Velocity = vector.MakeVector2(0.01, 0.01)

				server.state.SetAgentState(
					agentuuid,
					agentstate,
				)
			}
		}
	}
}

func processMovingObjectsCollisions(server *Server, movements []*movementState) []collision {
	return nil

	// collisions := make([]collision, 0)
	// collisionsMutex := &sync.Mutex{}

	// wait := &sync.WaitGroup{}
	// wait.Add(len(movements))

	// for _, movement := range movements {
	// 	go func(movement *movementState) {

	// 		bbRegion, err := getTrajectoryBoundingBox(
	// 			movement.Before.Position, movement.Before.Radius,
	// 			movement.After.Position, movement.After.Radius,
	// 		)
	// 		if err != nil {
	// 			utils.Debug("arena-server-updatestate", "Error in processMovingObjectsCollisions: could not define bbRegion in moving rTree")
	// 			return
	// 		}

	// 		matchingObstacles := server.state.MapMemoization.RtreeMoving.SearchIntersect(bbRegion, func(results []rtreego.Spatial, object rtreego.Spatial) (refuse, abort bool) {
	// 			if iteratedmovement, ok := object.(*movementState); ok {
	// 				if movementStateComparator(iteratedmovement, movement) {
	// 					// Object overlaps itself; refusing in collisions
	// 					return true, false
	// 				}
	// 			}

	// 			return false, false
	// 		})

	// 		if len(matchingObstacles) > 0 { // 1:  the object overlaps itself
	// 			collisionsMutex.Lock()
	// 			for _, matchingObstacle := range matchingObstacles {
	// 				if matchingMovement, ok := matchingObstacle.(*movementState); ok {
	// 					collisions = append(collisions, collision{
	// 						ownerType: movement.Type,
	// 						ownerID:   movement.ID,
	// 						otherType: matchingMovement.Type,
	// 						otherID:   matchingMovement.ID,
	// 						point:     matchingMovement.After.Position,
	// 						timeBegin: 0,
	// 						timeEnd:   1,
	// 					})
	// 				}
	// 			}
	// 			collisionsMutex.Unlock()
	// 		}

	// 		wait.Done()
	// 	}(movement)
	// }

	// return collisions
}

func processProjectileObstacleCollisions(server *Server, before map[uuid.UUID]movingObjectTemporaryState) []collision {

	collisions := make([]collision, 0)
	collisionsMutex := &sync.Mutex{}

	wait := &sync.WaitGroup{}
	wait.Add(len(before))

	for projectileid, beforestate := range before {
		go func(beforestate movingObjectTemporaryState, projectileid uuid.UUID) {

			projectile := server.state.GetProjectile(projectileid)

			afterstate := movingObjectTemporaryState{
				Position: projectile.Position,
				Velocity: projectile.Velocity,
				Radius:   projectile.Radius,
			}

			processMovingObjectObstacleCollision(server, beforestate, afterstate, []int{state.GeometryObjectType.ObstacleGround}, func(collisionPoint vector.Vector2, other *state.GeometryObject) {

				var tBegin float64 = 0.0
				var tEnd float64 = 1.0

				// distance du point de collision à la position initiale
				distTravel := afterstate.Position.Sub(beforestate.Position).Mag()
				if distTravel > 0 {
					distCollision := collisionPoint.Sub(beforestate.Position).Mag()
					tBegin = distCollision / distTravel // l'impact avec un obstacle est immédiat, pas de déplacement sur le point de collision
					tEnd = distCollision / distTravel
				}

				collisionsMutex.Lock()
				collisions = append(collisions, collision{
					ownerType: state.GeometryObjectType.Projectile,
					ownerID:   projectileid.String(),
					otherType: other.Type,
					otherID:   other.ID,
					point:     collisionPoint,
					timeBegin: tBegin,
					timeEnd:   tEnd,
				})
				collisionsMutex.Unlock()
			})

			wait.Done()

		}(beforestate, projectileid)
	}

	wait.Wait()

	return collisions
}

func processAgentObstacleCollisions(server *Server, before map[uuid.UUID]movingObjectTemporaryState) []collision {

	collisions := make([]collision, 0)
	collisionsMutex := &sync.Mutex{}

	wait := &sync.WaitGroup{}
	wait.Add(len(before))

	for agentid, beforestate := range before {
		go func(beforestate movingObjectTemporaryState, agentid uuid.UUID) {
			agentstate := server.state.GetAgentState(agentid)

			afterstate := movingObjectTemporaryState{
				Position: agentstate.Position,
				Velocity: agentstate.Velocity,
				Radius:   agentstate.Radius,
			}

			processMovingObjectObstacleCollision(server, beforestate, afterstate, nil, func(collisionPoint vector.Vector2, other *state.GeometryObject) {

				var tBegin float64 = 0.0
				var tEnd float64 = 1.0

				// distance du point de collision à la position initiale
				distTravel := afterstate.Position.Sub(beforestate.Position).Mag()
				if distTravel > 0 {
					distCollision := collisionPoint.Sub(beforestate.Position).Mag()
					tBegin = distCollision / distTravel // l'impact avec un obstacle est immédiat, pas de déplacement sur le point de collision
					tEnd = distCollision / distTravel
				}

				collisionsMutex.Lock()
				collisions = append(collisions, collision{
					ownerType: state.GeometryObjectType.Agent,
					ownerID:   agentid.String(),
					otherType: other.Type,
					otherID:   other.ID,
					point:     collisionPoint,
					timeBegin: tBegin,
					timeEnd:   tEnd,
				})
				collisionsMutex.Unlock()
			})

			wait.Done()

		}(beforestate, agentid)
	}

	wait.Wait()

	return collisions
}

type collisionWrapper struct {
	Point    vector.Vector2
	Obstacle *state.GeometryObject
}

func processMovingObjectObstacleCollision(server *Server, beforeState, afterState movingObjectTemporaryState, geotypesIgnored []int, collisionhandler func(collision vector.Vector2, geoObject *state.GeometryObject)) {

	bbRegion, err := getTrajectoryBoundingBox(beforeState.Position, beforeState.Radius, afterState.Position, afterState.Radius)
	if err != nil {
		utils.Debug("arena-server-updatestate", "Error in processMovingObjectObstacleCollision: could not define bbRegion in obstacle rTree")
		return
	}

	matchingObstacles := server.state.MapMemoization.RtreeObstacles.SearchIntersect(bbRegion)

	if len(matchingObstacles) > 0 {

		// Fine collision checking

		// We determine the surface occupied by the object on it's path
		// * Corresponds to a "pill", where the two ends are the bounding circles occupied by the agents (position before the move and position after the move)
		// * And the surface in between is defined the lines between the left and the right tangents of these circles
		//
		// * We then have to test collisions with the end circle
		//

		centerEdge := vector.MakeSegment2(beforeState.Position, afterState.Position)
		beforeDiameterSegment := centerEdge.OrthogonalToACentered().SetLengthFromCenter(beforeState.Radius * 2)
		afterDiameterSegment := centerEdge.OrthogonalToBCentered().SetLengthFromCenter(afterState.Radius * 2)

		beforeDiameterSegmentLeftPoint, beforeDiameterSegmentRightPoint := beforeDiameterSegment.Get()
		afterDiameterSegmentLeftPoint, afterDiameterSegmentRightPoint := afterDiameterSegment.Get()

		leftEdge := vector.MakeSegment2(beforeDiameterSegmentLeftPoint, afterDiameterSegmentLeftPoint)
		rightEdge := vector.MakeSegment2(beforeDiameterSegmentRightPoint, afterDiameterSegmentRightPoint)

		edgesToTest := []vector.Segment2{
			leftEdge,
			centerEdge,
			rightEdge,
		}

		collisions := make([]collisionWrapper, 0)

		for _, matchingObstacle := range matchingObstacles {
			geoObject := matchingObstacle.(*state.GeometryObject)
			if geotypesIgnored != nil && arrayContainsGeotype(geoObject.Type, geotypesIgnored) {
				continue
			}

			circleCollisions := trigo.LineCircleIntersectionPoints(
				geoObject.PointA,
				geoObject.PointB,
				afterState.Position,
				afterState.Radius,
			)

			for _, circleCollision := range circleCollisions {
				collisions = append(collisions, collisionWrapper{
					Point:    circleCollision,
					Obstacle: geoObject,
				})
			}

			for _, edge := range edgesToTest {
				point1, point2 := edge.Get()
				if collisionPoint, intersects, colinear, _ := trigo.IntersectionWithLineSegment(
					geoObject.PointA,
					geoObject.PointB,
					point1,
					point2,
				); intersects && !colinear {
					collisions = append(collisions, collisionWrapper{
						Point:    collisionPoint,
						Obstacle: geoObject,
					})
				}
			}
		}

		if len(collisions) > 0 {

			minDist := -1.0
			var firstCollision *collisionWrapper
			for _, collision := range collisions {
				thisDist := collision.Point.Sub(beforeState.Position).Mag()
				if minDist < 0 || minDist > thisDist {
					minDist = thisDist
					firstCollision = &collision
				}
			}

			backoffDistance := beforeState.Radius + 0.001
			nextPoint := centerEdge.Vector2().SetMag(minDist - backoffDistance).Add(beforeState.Position)

			if !isInsideGroundSurface(server, nextPoint) {

				// backtracking position to last not outside
				backsteps := 10
				railRel := afterState.Position.Sub(beforeState.Position)
				for k := 1; k <= backsteps; k++ {
					nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
					if isInsideGroundSurface(server, nextPointRel.Add(beforeState.Position)) {
						collisionhandler(nextPointRel.Add(beforeState.Position), firstCollision.Obstacle)
						return
					}
				}

				collisionhandler(beforeState.Position, firstCollision.Obstacle)

			} else {
				if isInsideCollisionMesh(server, nextPoint) {
					if isInsideCollisionMesh(server, beforeState.Position) {
						// moving it outside the mesh !!
						railRel := afterState.Position.Sub(beforeState.Position)
						railRel = railRel.Sub(railRel.SetMag(0.1))
						collisionhandler(railRel.Add(beforeState.Position), firstCollision.Obstacle)

					} else {
						// backtracking position to last not in obstacle
						backsteps := 10
						railRel := afterState.Position.Sub(beforeState.Position)
						railRel = railRel.Sub(railRel.SetMag(0.05))
						for k := 1; k <= backsteps; k++ {
							nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
							if !isInsideCollisionMesh(server, nextPointRel.Add(beforeState.Position)) {
								collisionhandler(nextPointRel.Add(beforeState.Position), firstCollision.Obstacle)
								return
							}
						}

						collisionhandler(beforeState.Position, firstCollision.Obstacle)
					}

				} else {
					collisionhandler(nextPoint, firstCollision.Obstacle)
				}
			}

		}
	}
}

func isInsideGroundSurface(server *Server, point vector.Vector2) bool {

	px, py := point.Get()

	bb, _ := rtreego.NewRect([]float64{px - 0.005, py - 0.005}, []float64{0.01, 0.01})
	matchingTriangles := server.state.MapMemoization.RtreeSurface.SearchIntersect(bb)

	if len(matchingTriangles) == 0 {
		return false
	}

	// On vérifie que le point est bien dans un des triangles
	for _, spatial := range matchingTriangles {
		triangle := spatial.(*state.TriangleRtreeWrapper)
		if trigo.PointIsInTriangle(point, triangle.Points[0], triangle.Points[1], triangle.Points[2]) {
			return true
		}
	}

	return false
}

func isInsideCollisionMesh(server *Server, point vector.Vector2) bool {

	px, py := point.Get()

	bb, _ := rtreego.NewRect([]float64{px - 0.005, py - 0.005}, []float64{0.01, 0.01})
	matchingTriangles := server.state.MapMemoization.RtreeCollisions.SearchIntersect(bb)

	if len(matchingTriangles) == 0 {
		return false
	}

	// On vérifie que le point est bien dans un des triangles
	for _, spatial := range matchingTriangles {
		triangle := spatial.(*state.TriangleRtreeWrapper)
		if trigo.PointIsInTriangle(point, triangle.Points[0], triangle.Points[1], triangle.Points[2]) {
			return true
		}
	}

	return false
}

func arrayContainsGeotype(needle int, haystack []int) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func getGeometryObjectBoundingBox(position vector.Vector2, radius float64) (topleft vector.Vector2, bottomright vector.Vector2) {
	x, y := position.Get()
	return vector.MakeVector2(x-radius, y-radius), vector.MakeVector2(x+radius, y+radius)
}

func getTrajectoryBoundingBox(beginPoint vector.Vector2, beginRadius float64, endPoint vector.Vector2, endRadius float64) (*rtreego.Rect, error) {
	beginTopLeft, beginBottomRight := getGeometryObjectBoundingBox(beginPoint, beginRadius)
	endTopLeft, endBottomRight := getGeometryObjectBoundingBox(endPoint, endRadius)

	bbTopLeft, bbDimensions := state.GetBoundingBox([]vector.Vector2{beginTopLeft, beginBottomRight, endTopLeft, endBottomRight})
	bbRegion, err := rtreego.NewRect(bbTopLeft, bbDimensions)
	if err != nil {
		return nil, errors.New("Error in getTrajectoryBoundingBox: could not define bbRegion in rTree")
	}

	return bbRegion, nil
}

type movementState struct {
	Type   int
	ID     string
	Before movingObjectTemporaryState
	After  movingObjectTemporaryState
	Rect   *rtreego.Rect
}

func (geobj movementState) Bounds() *rtreego.Rect {
	return geobj.Rect
}

func movementStateComparator(obj1, obj2 rtreego.Spatial) bool {
	sp1 := obj1.(*movementState)
	sp2 := obj2.(*movementState)

	return sp1.Type == sp2.Type && sp1.ID == sp2.ID
}
