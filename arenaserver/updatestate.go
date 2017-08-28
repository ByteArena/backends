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

		//show := spew.ConfigState{MaxDepth: 5, Indent: "    "}
		movements := make([]*movementState, 0)

		///////////////////////////////////////////////////////////////////////////
		// Moving objects collisions
		///////////////////////////////////////////////////////////////////////////

		// Indexing agents trajectories in rtree
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

			//show.Dump(bbRegion)

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

			movements = append(movements, &movementState{
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

		colls := processMovingObjectsCollisions(server, movements, rtMoving)
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

		if coll.ownerType == state.GeometryObjectType.Projectile {
			projuuid, _ := uuid.FromString(coll.ownerID)
			proj := server.state.GetProjectile(projuuid)
			if proj.AgentEmitterId.String() == coll.otherID {
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
		switch coll.ownerType {
		case state.GeometryObjectType.Projectile:
			{
				projectileuuid, _ := uuid.FromString(coll.ownerID)

				projectile := server.state.GetProjectile(projectileuuid)
				projectile.Position = coll.point
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

func processMovingObjectsCollisions(server *Server, movements []*movementState, rtMoving *rtreego.Rtree) []collision {

	//show := spew.ConfigState{MaxDepth: 5, Indent: "    "}

	collisions := make([]collision, 0)
	collisionsMutex := &sync.Mutex{}

	wait := &sync.WaitGroup{}
	wait.Add(len(movements))

	for _, movement := range movements {
		go func(movement *movementState) {

			matchingObjects := rtMoving.SearchIntersect(movement.Rect, func(results []rtreego.Spatial, object rtreego.Spatial) (refuse, abort bool) {
				return object == movement, false // avoid object overlapping itself
			})

			if len(matchingObjects) == 0 {
				wait.Done()
				return
			}

			//show.Dump("COLLISIONS", matchingObjects)
			//dump, _ := json.Marshal(matchingObjects)
			//log.Println("COARSE COLLISIONS", string(dump))

			fineCollisionMovingMovingChecking(server, movement.Before, movement.After, matchingObjects, nil, func(collisionPoint vector.Vector2, other finelyCollisionable) {
				// if other.GetType() == state.GeometryObjectType.Projectile {
				// 	// DEBUG; REMOVE THIS EARLY EXIT CONDITION
				// 	return
				// }

				// distance à la position initiale du point de collision des deux segments de trajectoire
				distTravel := movement.After.Position.Sub(movement.Before.Position).Mag()
				if distTravel > 0 {
					distCollisionOwner := collisionPoint.Sub(movement.Before.Position).Mag()

					tBeginOwner := (distCollisionOwner - movement.After.Radius) / distTravel
					tEndOwner := (distCollisionOwner + movement.After.Radius) / distTravel

					distCollisionOther := collisionPoint.Sub(other.GetPointA()).Mag()
					tBeginOther := (distCollisionOther - other.GetRadius()) / distTravel
					tEndOther := (distCollisionOther + other.GetRadius()) / distTravel

					if tEndOther < tBeginOwner || tEndOwner < tBeginOther {
						// 	// no time intersection, no collision !
					} else {
						log.Println("MOVING/MOVING COLLISION !")
						// they were at the same place at the same time in the tick ! Collision !
						collisionsMutex.Lock()
						collisions = append(collisions, collision{
							ownerType: movement.Type,
							ownerID:   movement.ID,
							otherType: other.GetType(),
							otherID:   other.GetID(),
							point:     collisionPoint,
							timeBegin: tBeginOwner - 0.001,
							timeEnd:   tEndOwner + 0.001,
						})
						collisionsMutex.Unlock()
					}
				} else {
					collisionsMutex.Lock()
					collisions = append(collisions, collision{
						ownerType: movement.Type,
						ownerID:   movement.ID,
						otherType: other.GetType(),
						otherID:   other.GetID(),
						point:     collisionPoint,
						timeBegin: 0,
						timeEnd:   1,
					})
					collisionsMutex.Unlock()
				}
			})

			wait.Done()
		}(movement)
	}

	wait.Wait()

	return collisions
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

			processMovingObjectObstacleCollision(server, beforestate, afterstate, []int{state.GeometryObjectType.ObstacleGround}, func(collisionPoint vector.Vector2, other finelyCollisionable) {

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
					otherType: other.GetType(),
					otherID:   other.GetID(),
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

			processMovingObjectObstacleCollision(server, beforestate, afterstate, nil, func(collisionPoint vector.Vector2, other finelyCollisionable) {
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
					otherType: other.GetType(),
					otherID:   other.GetID(),
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
	Obstacle finelyCollisionable
}

type collisionHandlerFunc func(collision vector.Vector2, geoObject finelyCollisionable)

func processMovingObjectObstacleCollision(server *Server, beforeState, afterState movingObjectTemporaryState, geotypesIgnored []int, collisionhandler collisionHandlerFunc) {

	bbRegion, err := getTrajectoryBoundingBox(beforeState.Position, beforeState.Radius, afterState.Position, afterState.Radius)
	if err != nil {
		utils.Debug("arena-server-updatestate", "Error in processMovingObjectObstacleCollision: could not define bbRegion in obstacle rTree")
		return
	}

	matchingObstacles := server.state.MapMemoization.RtreeObstacles.SearchIntersect(bbRegion)

	if len(matchingObstacles) > 0 {
		// fine collision checking
		fineCollisionChecking(server, beforeState, afterState, matchingObstacles, geotypesIgnored, collisionhandler)
	}
}

func fineCollisionChecking(server *Server, beforeState, afterState movingObjectTemporaryState, matchingObstacles []rtreego.Spatial, geotypesIgnored []int, collisionhandler collisionHandlerFunc) {
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
		geoObject := matchingObstacle.(finelyCollisionable)
		if geotypesIgnored != nil && arrayContainsGeotype(geoObject.GetType(), geotypesIgnored) {
			continue
		}

		if !geoObject.GetPointA().Equals(geoObject.GetPointB()) {
			circleCollisions := trigo.LineCircleIntersectionPoints(
				geoObject.GetPointA(),
				geoObject.GetPointB(),
				afterState.Position,
				afterState.Radius,
			)
			//log.Println(circleCollisions, geoObject.GetPointA(), geoObject.GetPointB(), afterState.Position, afterState.Radius)

			for _, circleCollision := range circleCollisions {
				collisions = append(collisions, collisionWrapper{
					Point:    circleCollision,
					Obstacle: geoObject,
				})
			}
		}

		for _, edge := range edgesToTest {
			point1, point2 := edge.Get()
			if collisionPoint, intersects, colinear, _ := trigo.IntersectionWithLineSegment(
				geoObject.GetPointA(),
				geoObject.GetPointB(),
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
		for _, coll := range collisions {
			thisDist := coll.Point.Sub(beforeState.Position).Mag()
			if minDist < 0 || minDist > thisDist {
				minDist = thisDist
				firstCollision = &coll
			}
		}

		nextPoint := firstCollision.Point

		backoffDistance := beforeState.Radius
		if minDist > backoffDistance {
			nextPoint = afterState.Position.Sub(beforeState.Position).SetMag(minDist - backoffDistance).Add(beforeState.Position)
		} else {
			collisionhandler(beforeState.Position, firstCollision.Obstacle)
			return
		}

		if isInsideGroundSurface(server, nextPoint) {
			if isInsideCollisionMesh(server, nextPoint) {
				if isInsideCollisionMesh(server, beforeState.Position) {
					// moving it outside the mesh !!
					railRel := afterState.Position.Sub(beforeState.Position)
					railRel = railRel.Sub(railRel.SetMag(0.1))
					//log.Println("TWOOOOOOOOOOOOOOOOOOOOO", firstCollision.Obstacle.GetType())
					collisionhandler(railRel.Add(beforeState.Position), firstCollision.Obstacle)

				} else {
					// backtracking position to last not in obstacle
					backsteps := 10
					railRel := nextPoint.Sub(beforeState.Position)
					railRel = railRel.Sub(railRel.SetMag(0.05))
					for k := 1; k <= backsteps; k++ {
						nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
						if !isInsideCollisionMesh(server, nextPointRel.Add(beforeState.Position)) {
							//log.Println("THREEEEEEEEEEEEEEEEEEEEEEEE", firstCollision.Obstacle.GetType())
							collisionhandler(nextPointRel.Add(beforeState.Position), firstCollision.Obstacle)
							return
						}
					}

					//log.Println("FOUUUUUUUUUUUUUUUR", firstCollision.Obstacle.GetType())
					collisionhandler(beforeState.Position, firstCollision.Obstacle)
				}

			} else {

				if firstCollision.Obstacle.GetType() == state.GeometryObjectType.ObstacleObject {
					backsteps := 10
					railRel := nextPoint.Sub(beforeState.Position)
					railRel = railRel.Sub(railRel.SetMag(0.05))
					for k := 0; k <= backsteps; k++ {
						nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
						if !isInsideCollisionMesh(server, nextPointRel.Add(beforeState.Position)) {
							//log.Println("FIIIIIIIIIIIVE", firstCollision.Obstacle.GetType())
							collisionhandler(nextPointRel.Add(beforeState.Position), firstCollision.Obstacle)
							return
						}
					}
				}

				//log.Println("SIIIIIIIIIIIIIIIIIIIIIX", firstCollision.Obstacle.GetType())
				collisionhandler(nextPoint, firstCollision.Obstacle)
			}

		} else {

			// backtracking position to last not outside
			backsteps := 10
			railRel := nextPoint.Sub(beforeState.Position)
			for k := 1; k <= backsteps; k++ {
				nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
				if isInsideGroundSurface(server, nextPointRel.Add(beforeState.Position)) {
					//log.Println("ZEEEEEEEEEEEEEEEROOOOOOOOOOOOO", firstCollision.Obstacle.GetType())
					collisionhandler(nextPointRel.Add(beforeState.Position), firstCollision.Obstacle)
					return
				}
			}

			//log.Println("OOOOOOOOOOOOOOOOOOONE", firstCollision.Obstacle.GetType())
			collisionhandler(beforeState.Position, firstCollision.Obstacle)
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

func getGeometryObjectBoundingBox(position vector.Vector2, radius float64) (bottomLeft vector.Vector2, topRight vector.Vector2) {
	x, y := position.Get()
	return vector.MakeVector2(x-radius, y-radius), vector.MakeVector2(x+radius, y+radius)
}

func getTrajectoryBoundingBox(beginPoint vector.Vector2, beginRadius float64, endPoint vector.Vector2, endRadius float64) (*rtreego.Rect, error) {
	beginBottomLeft, beginTopRight := getGeometryObjectBoundingBox(beginPoint, beginRadius)
	endBottomLeft, endTopRight := getGeometryObjectBoundingBox(endPoint, endRadius)

	bbTopLeft, bbDimensions := state.GetBoundingBox([]vector.Vector2{beginBottomLeft, beginTopRight, endBottomLeft, endTopRight})

	//show := spew.ConfigState{MaxDepth: 5, Indent: "    "}

	bbRegion, err := rtreego.NewRect(bbTopLeft, bbDimensions)
	if err != nil {
		return nil, errors.New("Error in getTrajectoryBoundingBox: could not define bbRegion in rTree")
	}

	// fmt.Println("----------------------------------------------------------------")
	// show.Dump(bbTopLeft, bbDimensions, bbRegion)
	// fmt.Println("----------------------------------------------------------------")

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

func (geobj *movementState) GetPointA() vector.Vector2 {
	return geobj.Before.Position
}

func (geobj *movementState) GetPointB() vector.Vector2 {
	return geobj.After.Position
}

func (geobj *movementState) GetRadius() float64 {
	return geobj.After.Radius
}

func (geobj *movementState) GetType() int {
	return geobj.Type
}

func (geobj *movementState) GetID() string {
	return geobj.ID
}

func movementStateComparator(obj1, obj2 rtreego.Spatial) bool {
	sp1 := obj1.(*movementState)
	sp2 := obj2.(*movementState)

	return sp1.Type == sp2.Type && sp1.ID == sp2.ID
}

type finelyCollisionable interface {
	GetPointA() vector.Vector2
	GetPointB() vector.Vector2
	GetRadius() float64
	GetType() int
	GetID() string
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func makePoly(centerA, centerB vector.Vector2, radiusA, radiusB float64) []vector.Vector2 {
	hasMoved := !centerA.Equals(centerB)
	var poly []vector.Vector2 = nil

	if hasMoved {

		AB := vector.MakeSegment2(centerA, centerB)
		// on détermine les 4 points formant le rectangle orienté définissant la trajectoire de l'object en movement

		polyASide := AB.OrthogonalToACentered().SetLengthFromCenter(radiusA * 2) // si AB vertical, A à gauche, B à droite
		polyBSide := AB.OrthogonalToBCentered().SetLengthFromCenter(radiusB * 2) // si AB vertical, A à gauche, B à droite

		/*

			B2*--------------------*A2
			  |                    |
			B *                    * A
			  |                    |
			B1*--------------------*A1

		*/

		polyA1, polyA2 := polyASide.Get()
		polyB1, polyB2 := polyBSide.Get()

		poly = []vector.Vector2{polyA1, polyA2, polyB2, polyB1}
	}

	return poly
}

func collideOrientedRectangles(polyOne, polyTwo []vector.Vector2) []vector.Vector2 {

	points := make([]vector.Vector2, 0)

	if !trigo.DoClosedConvexPolygonsIntersect(polyOne, polyTwo) {
		return points
	}

	/*

	 C*--------------------*B
	  |                    |
	  |                    |
	  |                    |
	 D*--------------------*A

	*/

	for i := 0; i < 4; i++ {
		polyOneEdge := vector.MakeSegment2(polyOne[i], polyOne[(i+1)%4])
		for j := 0; j < 4; j++ {
			polyTwoEdge := vector.MakeSegment2(polyOne[j], polyOne[(j+1)%4])
			if collisionPoint, intersects, colinear, _ := trigo.SegmentIntersectionWithLineSegment(
				polyOneEdge,
				polyTwoEdge,
			); intersects && !colinear {
				points = append(points, collisionPoint)
			}
		}
	}

	if len(points) == 0 {
		// one is inside the other
		// compute the area and send the points of the smallest one
		polyOneHeight := vector.MakeSegment2(polyOne[0], polyOne[1])
		polyOneWidth := vector.MakeSegment2(polyOne[1], polyOne[2])

		polyTwoHeight := vector.MakeSegment2(polyTwo[0], polyTwo[1])
		polyTwoWidth := vector.MakeSegment2(polyTwo[1], polyTwo[2])

		polyOneAreaSq := polyOneWidth.LengthSq() * polyOneHeight.LengthSq()
		polyTwoAreaSq := polyTwoWidth.LengthSq() * polyTwoHeight.LengthSq()

		if polyTwoAreaSq < polyOneAreaSq {
			return polyTwo
		}

		return polyOne
	}

	return points
}

func collidePolyCircle(poly []vector.Vector2, center vector.Vector2, radius float64) []vector.Vector2 {

	points := make([]vector.Vector2, 0)

	for i := 0; i < 4; i++ {
		points = append(points, trigo.LineCircleIntersectionPoints(poly[i], poly[(i+1)%4], center, radius)...)
	}

	if len(points) == 0 {
		// pas de collision avec les lignes
		// on vérifie si le polygone est inscrit au cercle
		for i := 0; i < 4; i++ {
			if trigo.PointIsInCircle(poly[i], center, radius) {
				// Si l'un des points du polygone est dans le cercle, comme aucune de ses lignes n'intersecte le cercle, c'est qu'il y est inscrit
				return poly
			}
		}

		// on vérifie si le cercle est inscrit au poly
		if trigo.PointIsInTriangle(center, poly[0], poly[1], poly[2]) || trigo.PointIsInTriangle(center, poly[2], poly[3], poly[0]) {
			return []vector.Vector2{center}
		}

	}

	return points
}

func collideCirclesCircles(colliderCenterA vector.Vector2, colliderRadiusA float64, collideeCenterA vector.Vector2, collideeRadiusA float64, colliderCenterB vector.Vector2, colliderRadiusB float64, collideeCenterB vector.Vector2, collideeRadiusB float64) []vector.Vector2 {

	points := make([]vector.Vector2, 0)

	// ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
	// ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA

	// ColliderCircleA/CollideeCircleA
	intersections, firstContainsSecond, secondContainsFirst := trigo.CircleCircleIntersectionPoints(colliderCenterA, colliderRadiusA, collideeCenterA, collideeRadiusA)
	if len(intersections) > 0 {
		points = append(points, intersections...)
	} else if firstContainsSecond {
		points = append(points, collideeCenterA)
	} else if secondContainsFirst {
		points = append(points, colliderCenterA)
	}

	// ColliderCircleB/CollideeCircleB
	intersections, firstContainsSecond, secondContainsFirst = trigo.CircleCircleIntersectionPoints(colliderCenterB, colliderRadiusB, collideeCenterB, collideeRadiusB)
	if len(intersections) > 0 {
		points = append(points, intersections...)
	} else if firstContainsSecond {
		points = append(points, collideeCenterB)
	} else if secondContainsFirst {
		points = append(points, colliderCenterB)
	}

	// ColliderCircleA/CollideeCircleB
	intersections, firstContainsSecond, secondContainsFirst = trigo.CircleCircleIntersectionPoints(colliderCenterA, colliderRadiusA, collideeCenterB, collideeRadiusB)
	if len(intersections) > 0 {
		points = append(points, intersections...)
	} else if firstContainsSecond {
		points = append(points, collideeCenterB)
	} else if secondContainsFirst {
		points = append(points, colliderCenterA)
	}

	// ColliderCircleB/CollideeCircleA
	intersections, firstContainsSecond, secondContainsFirst = trigo.CircleCircleIntersectionPoints(colliderCenterB, colliderRadiusB, collideeCenterA, collideeRadiusA)
	if len(intersections) > 0 {
		points = append(points, intersections...)
	} else if firstContainsSecond {
		points = append(points, collideeCenterA)
	} else if secondContainsFirst {
		points = append(points, colliderCenterB)
	}

	return points
}

func fineCollisionMovingMovingChecking(server *Server, colliderBeforeState, colliderAfterState movingObjectTemporaryState, matchingObstacles []rtreego.Spatial, geotypesIgnored []int, collisionhandler collisionHandlerFunc) {

	// Il faut vérifier l'intersection des (polygones + end circles) de trajectoire de Collider et de Collidee

	// On détermine les end circles de la trajectoire du collider
	colliderCenterA := colliderBeforeState.Position
	colliderRadiusA := colliderBeforeState.Radius

	colliderCenterB := colliderAfterState.Position
	colliderRadiusB := colliderAfterState.Radius

	colliderPoly := makePoly(colliderCenterA, colliderCenterB, colliderRadiusA, colliderRadiusB)

	collisions := make([]collisionWrapper, 0)

	for _, matchingObstacle := range matchingObstacles {
		points := make([]vector.Vector2, 0)

		geoObject := matchingObstacle.(finelyCollisionable)

		collideeCenterA := geoObject.GetPointA()
		collideeRadiusA := geoObject.GetRadius()

		collideeCenterB := geoObject.GetPointB()
		collideeRadiusB := geoObject.GetRadius()

		collideePoly := makePoly(collideeCenterA, collideeCenterB, collideeRadiusA, collideeRadiusB)

		/*
			Si colliderPoly && collideePoly, check:
				* ColliderPoly/CollideePoly
				* ColliderPoly/CollideeCircleA, ColliderPoly/CollideeCircleB
				* ColliderCircleA/CollideePoly, ColliderCircleB/CollideePoly
				* ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
				* ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA

			Si colliderPoly && !collideePoly, check:
				* ColliderPoly/CollideeCircleA, ColliderPoly/CollideeCircleB
				* ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
				* ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA

			Si !colliderPoly && collideePoly, check:
				* ColliderCircleA/CollideePoly, ColliderCircleB/CollideePoly
				* ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
				* ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA

			Si !colliderPoly && !collideePol, check:
				* ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
				* ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
		*/

		if colliderPoly != nil {
			if collideePoly != nil {
				/*
					Si colliderPoly && collideePoly, check:
						* ColliderPoly/CollideePoly
						* ColliderPoly/CollideeCircleA, ColliderPoly/CollideeCircleB
						* ColliderCircleA/CollideePoly, ColliderCircleB/CollideePoly
						* ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
						* ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
				*/

				// ColliderPoly/CollideePoly
				points = append(points, collideOrientedRectangles(colliderPoly, collideePoly)...)

				// ColliderPoly/CollideeCircleA, ColliderPoly/CollideeCircleB
				points = append(points, collidePolyCircle(colliderPoly, collideeCenterA, collideeRadiusA)...)
				points = append(points, collidePolyCircle(colliderPoly, collideeCenterB, collideeRadiusB)...)

				// ColliderCircleA/CollideePoly, ColliderCircleB/CollideePoly
				points = append(points, collidePolyCircle(collideePoly, colliderCenterA, colliderRadiusA)...)
				points = append(points, collidePolyCircle(collideePoly, colliderCenterB, colliderRadiusB)...)

				// ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
				// ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
				points = append(points, collideCirclesCircles(colliderCenterA, colliderRadiusA, collideeCenterA, collideeRadiusA, colliderCenterB, colliderRadiusB, collideeCenterB, collideeRadiusB)...)

			} else {
				/*
					Si colliderPoly && !collideePoly, check:
						* ColliderPoly/CollideeCircleA, ColliderPoly/CollideeCircleB
						* ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
						* ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
				*/

				// ColliderPoly/CollideeCircleA, ColliderPoly/CollideeCircleB
				points = append(points, collidePolyCircle(colliderPoly, collideeCenterA, collideeRadiusA)...)
				points = append(points, collidePolyCircle(colliderPoly, collideeCenterB, collideeRadiusB)...)

				// ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
				// ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
				points = append(points, collideCirclesCircles(colliderCenterA, colliderRadiusA, collideeCenterA, collideeRadiusA, colliderCenterB, colliderRadiusB, collideeCenterB, collideeRadiusB)...)
			}
		} else {
			if collideePoly != nil {
				/*
					Si !colliderPoly && collideePoly, check:
						* ColliderCircleA/CollideePoly, ColliderCircleB/CollideePoly
						* ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
						* ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
				*/

				// ColliderCircleA/CollideePoly, ColliderCircleB/CollideePoly
				points = append(points, collidePolyCircle(collideePoly, colliderCenterA, colliderRadiusA)...)
				points = append(points, collidePolyCircle(collideePoly, colliderCenterB, colliderRadiusB)...)

				// ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
				// ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
				points = append(points, collideCirclesCircles(colliderCenterA, colliderRadiusA, collideeCenterA, collideeRadiusA, colliderCenterB, colliderRadiusB, collideeCenterB, collideeRadiusB)...)
			} else {
				/*
					Si !colliderPoly && !collideePol, check:
						* ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
						* ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
				*/

				// ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
				// ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
				points = append(points, collideCirclesCircles(colliderCenterA, colliderRadiusA, collideeCenterA, collideeRadiusA, colliderCenterB, colliderRadiusB, collideeCenterB, collideeRadiusB)...)
			}
		}

		if len(points) > 0 {
			minDistSq := -1.0
			var firstCollision vector.Vector2
			for _, point := range points {
				thisDist := point.Sub(colliderBeforeState.Position).MagSq()
				if minDistSq < 0 || minDistSq > thisDist {
					minDistSq = thisDist
					firstCollision = point
				}
			}

			collisions = append(collisions, collisionWrapper{
				Point:    firstCollision,
				Obstacle: geoObject,
			})
		}

	}

	for _, collision := range collisions {
		collisionhandler(collision.Point, collision.Obstacle)
	}

}
