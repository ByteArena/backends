package arenaserver

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/bytearena/bytearena/common/utils/number"

	polyclip "github.com/akavel/polyclip-go"
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
	//show := spew.ConfigState{MaxDepth: 5, Indent: "    "}

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
		//show.Dump("RECEIVED HERE", colls)
		collisionsMutex.Lock()
		collisions = append(collisions, colls...)
		collisionsMutex.Unlock()
		wait.Done()
	}()

	wait.Wait()

	utils.Debug("collision-detection", fmt.Sprintf("Took %f ms; found %d collisions", time.Now().Sub(begin).Seconds()*1000, len(collisions)))

	// Ordering collisions along time (causality order)
	sort.Sort(CollisionByTimeAsc(collisions))

	//show.Dump("RESOLVED sorted", collisions)

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

				if projectile.AgentEmitterId.String() == coll.otherID {
					continue
				}

				//log.Println("PROJECTILE TOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOUCHED")

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
				if coll.otherType == state.GeometryObjectType.Projectile {
					projectileuuid, _ := uuid.FromString(coll.otherID)
					projectile := server.state.GetProjectile(projectileuuid)
					if projectile.AgentEmitterId.String() == agentuuid.String() {
						continue
					}
				}

				//log.Println("AGENT TOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOUCHED")

				agentstate := server.GetState().GetAgentState(agentuuid)
				agentstate.Position = coll.point
				agentstate.Orientation++
				agentstate.Velocity = vector.MakeNullVector2()
				agentstate.DebugNbHits++
				agentstate.DebugMsg = "HITS: " + strconv.Itoa(agentstate.DebugNbHits)

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

			fineCollisionMovingMovingChecking(server, movement, matchingObjects, nil, func(colls []collision) {
				collisionsMutex.Lock()
				collisions = append(collisions, colls...)
				collisionsMutex.Unlock()
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

	if !hasMoved {
		return nil
	}

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

	return []vector.Vector2{polyA1, polyA2, polyB2, polyB1}
}

func clipOrientedRectangles(polyOne, polyTwo []vector.Vector2) []vector.Vector2 {

	contourOne := make(polyclip.Contour, len(polyOne))
	for i, p := range polyOne {
		contourOne[i] = polyclip.Point{X: p.GetX(), Y: p.GetY()}
	}

	contourTwo := make(polyclip.Contour, len(polyTwo))
	for i, p := range polyTwo {
		contourTwo[i] = polyclip.Point{X: p.GetX(), Y: p.GetY()}
	}

	subject := polyclip.Polygon{contourOne}
	clipping := polyclip.Polygon{contourTwo}

	result := subject.Construct(polyclip.INTERSECTION, clipping)
	if len(result) == 0 || len(result[0]) < 3 {
		return nil
	}

	res := make([]vector.Vector2, len(result[0]))
	for i, p := range result[0] {
		res[i] = vector.MakeVector2(p.X, p.Y)
	}

	return res
}

func collideConstrainedCenterCircleWithPolygon(crossingPoly []vector.Vector2, centerSegment vector.Segment2, circleRadius float64) []vector.Segment2 {

	tangentsRadiuses := make([]vector.Segment2, 0) // point A: center position of object when colliding; point B: collision point

	/*
		TODO:
			collision du cercle du collider, dont le centre est sur la ligne centrale de trajectoire de l'objet
			avec le polygone obtenu par le clipping du rectangle orienté de trajectoire du collider avec le rectangle orienté de trajectoire du collidee.

			Les points de collision correspondent aux centres des cercles (aux positions des objets) tangents au début et à la fin du polygone.

			Pour identifier les faces du polygone sur lesquelles déterminer une tangente, il faut vérifier si la face du polygone testée est parallèle la ligne de centre de trajectoire de l'objet;
			si c'est le cas, il ne faut pas déterminer de tangente pour la face en question (utiliser pour ce test trigo.IntersectionWithLineSegment())
	*/

	// 1. On biaise les droites exactement verticales pour pouvoir toujours les décrire avec une équation affine
	colliderAffineSlope, _ /*colliderAffineYIntersect*/, colliderAffineIsVertical, _ /*colliderAffineVerticalX*/ := trigo.GetAffineEquationExpressedForY(centerSegment)
	if colliderAffineIsVertical {
		centerSegment = centerSegment.SetPointB(centerSegment.GetPointB().Add(vector.MakeVector2(0.0001, 0)))
		colliderAffineSlope, _ /*colliderAffineYIntersect*/, colliderAffineIsVertical, _ /*colliderAffineVerticalX*/ = trigo.GetAffineEquationExpressedForY(centerSegment)
		if colliderAffineIsVertical {
			// no collision will be processed !
			// may never happen
			return tangentsRadiuses
		}
		//panic("colliderAffineIsVertical !! what should we do ???")
	}

	// 2. On détermine les segments du polygone pour lesquels calculer une tangente

	type intersectingSegmentWrapper struct {
		point   vector.Vector2
		segment vector.Segment2
	}

	polyLen := len(crossingPoly)

	centerSegmentPointA, centerSegmentPointB := centerSegment.Get()
	touchingSegments := make([]intersectingSegmentWrapper, 0)

	for i, _ := range crossingPoly {
		p1 := crossingPoly[i]
		p2 := crossingPoly[(i+1)%polyLen]

		// Il s'agit bien d'une intersection de lignes et pas de segments
		// Car le collider peut entre en collision avec la ligne formée par le segment même si son centre n'entre pas en collision avec le segment (le radius du collider est non nul)
		if point, parallel := trigo.LinesIntersectionPoint(p1, p2, centerSegmentPointA, centerSegmentPointB); !parallel {
			touchingSegments = append(touchingSegments, intersectingSegmentWrapper{
				point:   point,
				segment: vector.MakeSegment2(p1, p2),
			})
		}
	}

	if len(touchingSegments) == 0 {
		// Pas d'intersection de la surface de trajectoire du collider avec celle du collidee
		// Ne devrait pas se produire, car ce cas est rendu impossible par le test 1, et par le fait que le polygone est une intersection des deux trajectoires considérées
		return tangentsRadiuses
	}

	// collision du cercle du collider, dont le centre est sur la ligne centrale de trajectoire de l'objet
	// avec le polygone obtenu par le clipping du rectangle orienté de trajectoire du collider avec le rectangle orienté de trajectoire du collidee.
	for _, touchingSegment := range touchingSegments {
		// On détermine l'équation affine de la droite passant entre les deux points du segment
		segmentAffineSlope, _ /*segmentAffineYIntersect*/, segmentAffineIsVertical, _ /*segmentAffineVerticalX*/ := trigo.GetAffineEquationExpressedForY(touchingSegment.segment)
		if segmentAffineIsVertical {
			//panic("segmentAffineIsVertical !! what should we do ???")
			touchingSegment.segment = touchingSegment.segment.SetPointB(touchingSegment.segment.GetPointB().Add(vector.MakeVector2(0.0001, 0)))
			segmentAffineSlope, _ /*colliderAffineYIntersect*/, segmentAffineIsVertical, _ /*colliderAffineVerticalX*/ = trigo.GetAffineEquationExpressedForY(touchingSegment.segment)
			if segmentAffineIsVertical {
				// no collision will be processed !
				// may never happen
				continue
			}
		}

		// le rayon pour la tangente recherchée (rt) est perpendiculaire au segment
		tangentRadiusSlope := -1 / segmentAffineSlope //perpendicular(y=ax+b) : y = -1/a
		var tangentRadiusSegment vector.Segment2

		// le centre du rayon (h, k) pour la tangente recherchée est le point d'intersection de rt et de la ligne de centre du collider
		if number.FloatEquals(tangentRadiusSlope, colliderAffineSlope) {
			// si le tangentRadiusSlope == colliderAffineSlope, la ligne de centre du collider est perpendiculaire au segment
			// le rayon au point de tangente est colinéaire à la ligne de centre du collider

			// On crée le segment depuis le début de la ligne de centre du collider jusqu'à sa collision avec la ligne (pas le segment) formée par le segment collidee
			tangentRadiusSegment = vector.MakeSegment2(centerSegmentPointA, touchingSegment.point).SetLengthFromB(circleRadius)
		} else {
			// il faut calculer le point d'intersection du segment perpendiculaire au collidee sur la ligne de centre du collider, et de longueur circleRadius
			// Utilisation du théorème de pythagore pour ce faire
			/*

					|\
				a	| \  c
					|  \
					|___\
					  b

					  c: ligne du collider
					  a: ligne du collidee
					  b: rayon de la tangente au cercle du collider

					On veut déterminer a
					On connaît b (circleRadius)
					On connaît la slope de l'angle ab (perpendiculaire)
					Il faut calculer l'angle ac
					Utiliser cet angle pour déterminer la slope (relation entre les longueurs a et c)
					Utiliser b, la slope de l'angle ac et le fait que le triangle soit rectangle ab pour calculer a

					Comme le triangle est rectangle, slopeac = a/b
					Donc: a = slopeac * b

					On utilise a pour déterminer b, et on recule depuis le point de collision de la ligne du collider de la longueur déterminée pour trouver le centre du cercle tangent
			*/

			absoluteAngleCRad := math.Atan2(
				centerSegmentPointA.GetY()-centerSegmentPointB.GetY(),
				centerSegmentPointA.GetX()-centerSegmentPointB.GetX(),
			)
			absoluteAngleARad := math.Atan2(
				touchingSegment.segment.GetPointA().GetY()-touchingSegment.segment.GetPointB().GetY(),
				touchingSegment.segment.GetPointA().GetX()-touchingSegment.segment.GetPointB().GetX(),
			)

			angleACRad := absoluteAngleCRad - absoluteAngleARad
			slopeAC := math.Tan(angleACRad)

			a := slopeAC * circleRadius // la distance entre le point d'intersection de la ligne du collider et la tangente du cercle de l'agent

			// on utilise a et b pour déterminer c
			// a2 + b2 = c2
			// c = sqrt(a2+b2)

			c := math.Sqrt(math.Pow(a, 2) + math.Pow(circleRadius, 2))

			// On recule depuis le point de collision de la ligne du collider de la longueur déterminée pour trouver le centre du cercle tangent
			colliderCollisionCourseSegment := vector.MakeSegment2(centerSegmentPointA, touchingSegment.point).SetLengthFromB(c)
			tangentCircleCenter := colliderCollisionCourseSegment.GetPointA()

			// on calcule l'intersection sur collidee du rayon de tangente passant en tangentCircleCenter
			// on connait le slope du rayon de tangente tangentRadiusSlope
			// on connait un point par lequel passe ce rayon tangentCircleCenter
			// on cherche l'équation affine de la ligne en question
			// y0 - y1 = m(x0 - x1)
			// y0 - y1 = m*x0 - m*x1
			// -y1 = m*x0 - m*x1 - y0
			// y1 = -1 * (m*x0 - m*x1 - y0)
			// y1 = (m*x1) + (y0 - m*x0)
			// y=ax+b
			// a=m
			// b=y0-m*x0

			tangentRadiusAffineYIntersect := tangentCircleCenter.GetY() - (tangentRadiusSlope * tangentCircleCenter.GetX())
			// on détermine un deuxième point de la ligne
			tangentRadiusPrimePointX := tangentCircleCenter.GetX() + 10
			tangentRadiusPrimePointY := tangentRadiusSlope*tangentRadiusPrimePointX + tangentRadiusAffineYIntersect

			// on détermine le point d'intersection du rayon de tangente
			tangentPoint, _ := trigo.LinesIntersectionPoint(
				touchingSegment.segment.GetPointA(), touchingSegment.segment.GetPointB(),
				tangentCircleCenter, vector.MakeVector2(tangentRadiusPrimePointX, tangentRadiusPrimePointY),
			)

			tangentRadiusSegment = vector.MakeSegment2(tangentCircleCenter, tangentPoint)
		}

		tangentsRadiuses = append(tangentsRadiuses, tangentRadiusSegment)

	}

	// Les points de collision correspondent aux centres des cercles (aux positions des objets) tangents au début et à la fin du polygone.

	return tangentsRadiuses
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
			); intersects {
				if !colinear {
					points = append(points, collisionPoint)
				} else {

					colinearIntersections := make([]vector.Vector2, 0)

					a1, a2 := polyOneEdge.Get()
					b1, b2 := polyTwoEdge.Get()
					if trigo.PointOnLineSegment(a1, b1, b2) {
						colinearIntersections = append(colinearIntersections, a1)
					}

					if trigo.PointOnLineSegment(a2, b1, b2) {
						colinearIntersections = append(colinearIntersections, a2)
					}

					if trigo.PointOnLineSegment(b1, a1, a2) {
						colinearIntersections = append(colinearIntersections, b1)
					}

					if trigo.PointOnLineSegment(b2, a1, a2) {
						colinearIntersections = append(colinearIntersections, b2)
					}

					if len(colinearIntersections) > 0 {
						centerOfMass, _ := trigo.ComputeCenterOfMass(colinearIntersections)
						points = append(points, centerOfMass)
					}
				}
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

func collideOrientedRectangleCircle(poly []vector.Vector2, center vector.Vector2, radius float64, trajectoryPointA, trajectoryPointB vector.Vector2, colliderRadius float64) []vector.Segment2 {

	collisionPositionSegments := make([]vector.Segment2, 0)

	points := make([]vector.Vector2, 0)

	// 2. On détermine les segments du polygone pour lesquels calculer une tangente

	trajectorySegment := vector.MakeSegment2(trajectoryPointA, trajectoryPointB)
	trajectoryLength := trajectorySegment.Length()

	type intersectingSegmentWrapper struct {
		point   vector.Vector2
		segment vector.Segment2
	}

	polyLen := len(poly)

	for i, _ := range poly {
		p1 := poly[i]
		p2 := poly[(i+1)%polyLen]

		// Il s'agit bien d'une intersection de lignes et pas de segments
		// Car le collider peut entre en collision avec la ligne formée par le segment même si son centre n'entre pas en collision avec le segment (le radius du collider est non nul)
		points = append(points, trigo.LineCircleIntersectionPoints(p1, p2, center, radius)...)
	}

	trajectorySlope, _ /*colliderAffineYIntersect*/, trajectoryIsVertical, _ /*colliderAffineVerticalX*/ := trigo.GetAffineEquationExpressedForY(trajectorySegment)
	if trajectoryIsVertical {
		trajectorySegment = trajectorySegment.SetPointB(trajectorySegment.GetPointB().Add(vector.MakeVector2(0.0001, 0)))
		trajectorySlope, _ /*YIntersect*/, trajectoryIsVertical, _ /*VerticalX*/ = trigo.GetAffineEquationExpressedForY(trajectorySegment)
		if trajectoryIsVertical {
			// no collision will be processed !
			// may never happen
			return []vector.Segment2{}
		}
	}

	orthoSlope := -1 / trajectorySlope //perpendicular(y=ax+b) : y = -1/a

	centerLinePoints := make([]vector.Vector2, 0)

	if len(points) == 0 {
		// Pas d'intersection de la surface de trajectoire du collider avec celle du cercle de collidee
		// Le cercle est peut-être trop petit

		if trigo.PointIsInTriangle(center, poly[0], poly[1], poly[2]) || trigo.PointIsInTriangle(center, poly[2], poly[3], poly[0]) {

			// on projette orthogonalement le centre du cercle sur la trajectoire du centre
			// on détermine l'orthogonale à la ligne de centre passant par le centre du cercle

			// on connaît un premier point sur l'orthogonale; on en cherche un deuxième
			// on détermine un deuxième point de la ligne
			// y1 = (m*x1) + (y0 - m*x0)

			orthoPrimePointX := center.GetX() + 10
			orthoPrimePointY := orthoSlope*orthoPrimePointX + (center.GetY() - orthoSlope*center.GetX())

			// on détermine le point d'intersection de l'orthogonale sur la ligne de trajectoire
			orthoCenterPoint, _ := trigo.LinesIntersectionPoint(
				trajectorySegment.GetPointA(), trajectorySegment.GetPointB(),
				center, vector.MakeVector2(orthoPrimePointX, orthoPrimePointY),
			)

			centerLinePoints = append(centerLinePoints, orthoCenterPoint)
		}
	} else {

		// Pour chaque point, on trouve son intersection avec la ligne centrale de trajectoire
		for _, p := range points {
			// on détermine l'orthogonale à la ligne de centre passant par le point en question

			// on connaît un premier point sur l'orthogonale; on en cherche un deuxième
			//orthoYIntersect := p.GetY() - (orthoSlope * p.GetX())

			// on détermine un deuxième point de la ligne
			// y1 = (m*x1) + (y0 - m*x0)

			orthoPrimePointX := p.GetX() + 10
			orthoPrimePointY := orthoSlope*orthoPrimePointX + (p.GetY() - orthoSlope*p.GetX())

			// on détermine le point d'intersection de l'orthogonale sur la ligne de trajectoire
			orthoCenterPoint, _ := trigo.LinesIntersectionPoint(
				trajectorySegment.GetPointA(), trajectorySegment.GetPointB(),
				p, vector.MakeVector2(orthoPrimePointX, orthoPrimePointY),
			)

			centerLinePoints = append(centerLinePoints, orthoCenterPoint)
		}
	}

	if len(centerLinePoints) >= 2 {
		// on identifie les distances min et max de projection des collisions sur la ligne de centre
		minDist := -1.0
		maxDist := -1.0
		for _, centerLinePoint := range centerLinePoints {
			dist := centerLinePoint.Sub(trajectorySegment.GetPointA()).Mag()
			if minDist < 0 || dist < minDist {
				minDist = dist
			}

			if maxDist < 0 || dist > maxDist {
				maxDist = dist
			}
		}

		if minDist > maxDist {
			minDist, maxDist = maxDist, minDist
		}

		minDist = minDist - colliderRadius
		maxDist = maxDist + colliderRadius

		if minDist < 0 {
			minDist = 0.001
		}

		if maxDist > trajectoryLength {
			maxDist = trajectoryLength
		}

		if minDist > maxDist {
			minDist, maxDist = maxDist, minDist
		}

		//log.Println("minDist", "maxDist", minDist, maxDist)

		collisionPositionSegments = append(collisionPositionSegments, trajectorySegment.SetLengthFromA(minDist))
		collisionPositionSegments = append(collisionPositionSegments, trajectorySegment.SetLengthFromA(maxDist))
	}

	return collisionPositionSegments
}

func collideCirclesCircles(colliderCenterA vector.Vector2, colliderRadiusA float64, collideeCenterA vector.Vector2, collideeRadiusA float64, colliderCenterB vector.Vector2, colliderRadiusB float64, collideeCenterB vector.Vector2, collideeRadiusB float64) []vector.Vector2 {

	points := make([]vector.Vector2, 0)

	if colliderCenterA.Equals(collideeCenterA) || colliderCenterA.Equals(collideeCenterB) {
		return []vector.Vector2{colliderCenterA}
	}

	if colliderCenterB.Equals(collideeCenterA) || colliderCenterB.Equals(collideeCenterB) {
		return []vector.Vector2{colliderCenterB}
	}

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

func fineCollisionMovingMovingChecking(server *Server, movement *movementState, matchingObstacles []rtreego.Spatial, geotypesIgnored []int, collisionhandler func(colls []collision)) {

	//show := spew.ConfigState{MaxDepth: 5, Indent: "    "}

	// Il faut vérifier l'intersection des (polygones + end circles) de trajectoire de Collider et de Collidee

	// On détermine les end circles de la trajectoire du collider
	colliderCenterA := movement.Before.Position
	colliderRadiusA := movement.Before.Radius

	colliderCenterB := movement.After.Position
	colliderRadiusB := movement.After.Radius

	colliderPoly := makePoly(colliderCenterA, colliderCenterB, colliderRadiusA, colliderRadiusB)

	for _, matchingObstacle := range matchingObstacles {

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
					TODO: collision du cercle du collider, dont le centre est sur la ligne centrale de trajectoire de l'objet
					avec le polygone obtenu par le clipping du rectangle orienté de trajectoire du collider avec le rectangle orienté de trajectoire du collidee
					Les points de collision correspondent aux centres des cercles (aux positions des objets) tangent auxdébut et à la fin du polygone
					Pour déterminer les faces du polygone sur lesquelles déterminer une tangente, il faut vérifier si la face du polygone testée est parallèle à la ligne de centre de trajectoire de l'objet
					Si c'est le cas, il ne faut pas déterminer de tangente pour la face en question (utiliser pour ce test trigo.IntersectionWithLineSegment())
				*/

				crossingPoly := clipOrientedRectangles(colliderPoly, collideePoly)
				if crossingPoly == nil {
					// Pas d'intersection des trajectoires
					continue
				}

				colliderCollidingPositions := collideConstrainedCenterCircleWithPolygon(
					crossingPoly,
					vector.MakeSegment2(colliderCenterA, colliderCenterB),
					colliderRadiusA,
				)

				collideeCollidingPositions := collideConstrainedCenterCircleWithPolygon(
					crossingPoly,
					vector.MakeSegment2(collideeCenterA, collideeCenterB),
					collideeRadiusA,
				)

				//log.Println(colliderCollidingPositions)
				if len(colliderCollidingPositions) == 2 && len(collideeCollidingPositions) == 2 {

					other := geoObject

					// order positions begin & end for both collider & collidee
					firstColliderPositionWhenColliding := colliderCollidingPositions[0].GetPointA()
					lastColliderPositionWhenColliding := colliderCollidingPositions[1].GetPointA()
					if firstColliderPositionWhenColliding.Sub(colliderCenterA).MagSq() > lastColliderPositionWhenColliding.Sub(colliderCenterA).MagSq() {
						firstColliderPositionWhenColliding, lastColliderPositionWhenColliding = lastColliderPositionWhenColliding, firstColliderPositionWhenColliding
					}

					firstCollideePositionWhenColliding := collideeCollidingPositions[0].GetPointA()
					lastCollideePositionWhenColliding := collideeCollidingPositions[1].GetPointA()
					if firstCollideePositionWhenColliding.Sub(other.GetPointA()).MagSq() > lastCollideePositionWhenColliding.Sub(other.GetPointA()).MagSq() {
						firstCollideePositionWhenColliding, lastCollideePositionWhenColliding = lastCollideePositionWhenColliding, firstCollideePositionWhenColliding
					}

					// distance à la position initiale du point de collision des deux segments de trajectoire
					distColliderTravel := colliderCenterB.Sub(colliderCenterA).Mag()
					tBeginCollider := 0.0
					tEndCollider := 1.0
					if distColliderTravel > 0 {
						distFirstPositionCollider := firstColliderPositionWhenColliding.Sub(colliderCenterA).Mag()
						distLastPositionCollider := lastColliderPositionWhenColliding.Sub(colliderCenterA).Mag()

						tBeginCollider = distFirstPositionCollider / distColliderTravel
						tEndCollider = distLastPositionCollider / distColliderTravel
					}

					distCollideeTravel := other.GetPointB().Sub(other.GetPointA()).Mag()
					tBeginCollidee := 0.0
					tEndCollidee := 1.0
					if distCollideeTravel > 0 {
						distFirstPositionCollidee := firstCollideePositionWhenColliding.Sub(other.GetPointA()).Mag()
						distLastPositionCollidee := lastCollideePositionWhenColliding.Sub(other.GetPointA()).Mag()

						tBeginCollidee = distFirstPositionCollidee / distCollideeTravel
						tEndCollidee = distLastPositionCollidee / distCollideeTravel
					}

					if tEndCollidee < tBeginCollider || tEndCollider < tBeginCollidee {
						// no time intersection, no collision !
					} else {
						//they were at the same place at the same time in the tick ! Collision !
						collisionhandler([]collision{{
							ownerType: movement.Type,
							ownerID:   movement.ID,
							otherType: other.GetType(),
							otherID:   other.GetID(),
							point:     firstColliderPositionWhenColliding,
							timeBegin: tBeginCollider,
							timeEnd:   tEndCollider,
						}})
					}
				}

				//continue

			}

			/*
				Si colliderPoly && !collideePoly, check:
					* ColliderPoly/CollideeCircleA, ColliderPoly/CollideeCircleB
					* ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
					* ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
			*/

			// Collider has moved
			// Determine the impact closest to colliderCenterA
			// And then determine the center position of the object when the collision happend

			// ColliderPoly/CollideeCircleA (no need for B, collidee has not moved)
			colliderCollidingPositions := collideOrientedRectangleCircle(colliderPoly, collideeCenterA, collideeRadiusA, colliderCenterA, colliderCenterB, colliderRadiusA)
			if len(colliderCollidingPositions) == 2 {

				//log.Println("UUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUUU", colliderCollidingPositions)

				other := geoObject

				// order positions begin & end for both collider & collidee
				firstColliderPositionWhenColliding := colliderCollidingPositions[0].GetPointA()
				lastColliderPositionWhenColliding := colliderCollidingPositions[1].GetPointA()
				if firstColliderPositionWhenColliding.Sub(colliderCenterA).MagSq() > lastColliderPositionWhenColliding.Sub(colliderCenterA).MagSq() {
					firstColliderPositionWhenColliding, lastColliderPositionWhenColliding = lastColliderPositionWhenColliding, firstColliderPositionWhenColliding
				}

				// distance à la position initiale du point de collision des deux segments de trajectoire
				distColliderTravel := colliderCenterB.Sub(colliderCenterA).Mag()
				tBeginCollider := 0.0
				tEndCollider := 1.0
				if distColliderTravel > 0 {
					distFirstPositionCollider := firstColliderPositionWhenColliding.Sub(colliderCenterA).Mag()
					distLastPositionCollider := lastColliderPositionWhenColliding.Sub(colliderCenterA).Mag()

					tBeginCollider = distFirstPositionCollider / distColliderTravel
					tEndCollider = distLastPositionCollider / distColliderTravel
				}

				tBeginCollidee := 0.0
				tEndCollidee := 1.0

				if tEndCollidee < tBeginCollider || tEndCollider < tBeginCollidee {
					// no time intersection, no collision !
				} else {
					colls := []collision{{
						ownerType: movement.Type,
						ownerID:   movement.ID,
						otherType: other.GetType(),
						otherID:   other.GetID(),
						point:     firstColliderPositionWhenColliding,
						timeBegin: tBeginCollider,
						timeEnd:   tEndCollider,
					}}
					if movement.Type == 3 && other.GetType() == 2 {
						//show.Dump("PROJECTILE INCOMING ON AGENT !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", colls)
					}
					//they were at the same place at the same time in the tick ! Collision !
					collisionhandler(colls)
				}

				continue
			}
		} else {
			if collideePoly != nil {
				/*
					Si !colliderPoly && collideePoly, check:
						* ColliderCircleA/CollideePoly, ColliderCircleB/CollideePoly
						* ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
						* ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
				*/

				// ColliderCircleA/CollideePoly (no need for B, collider has not moved)
				colliderCollidingPositions := collideOrientedRectangleCircle(collideePoly, collideeCenterB, collideeRadiusA, colliderCenterB, colliderCenterA, colliderRadiusA)
				if len(colliderCollidingPositions) == 2 {
					//log.Println("laaaaaaaaaaaaaaaaaaaa")

					other := geoObject

					// order positions begin & end for both collider & collidee
					firstColliderPositionWhenColliding := colliderCollidingPositions[0].GetPointA()
					lastColliderPositionWhenColliding := colliderCollidingPositions[1].GetPointA()
					if firstColliderPositionWhenColliding.Sub(colliderCenterA).MagSq() > lastColliderPositionWhenColliding.Sub(colliderCenterA).MagSq() {
						firstColliderPositionWhenColliding, lastColliderPositionWhenColliding = lastColliderPositionWhenColliding, firstColliderPositionWhenColliding
					}

					// distance à la position initiale du point de collision des deux segments de trajectoire
					distColliderTravel := colliderCenterB.Sub(colliderCenterA).Mag()
					tBeginCollider := 0.0
					tEndCollider := 1.0
					if distColliderTravel > 0 {
						distFirstPositionCollider := firstColliderPositionWhenColliding.Sub(colliderCenterA).Mag()
						distLastPositionCollider := lastColliderPositionWhenColliding.Sub(colliderCenterA).Mag()

						tBeginCollider = distFirstPositionCollider / distColliderTravel
						tEndCollider = distLastPositionCollider / distColliderTravel
					}

					tBeginCollidee := 0.0
					tEndCollidee := 1.0

					if tEndCollidee < tBeginCollider || tEndCollider < tBeginCollidee {
						// no time intersection, no collision !
					} else {
						//they were at the same place at the same time in the tick ! Collision !
						collisionhandler([]collision{{
							ownerType: movement.Type,
							ownerID:   movement.ID,
							otherType: other.GetType(),
							otherID:   other.GetID(),
							point:     firstColliderPositionWhenColliding,
							timeBegin: tBeginCollider,
							timeEnd:   tEndCollider,
						}})
					}
				}

				continue
			} else {
				// TODO HERE: handle Collider/Collidee separation ! right now, they center in on each other position, making it impossible for them to split
				/*
					Si !colliderPoly && !collideePol, check:
						* ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
						* ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
				*/

				// ColliderCircleA/CollideeCircleA, ColliderCircleB/CollideeCircleB
				// ColliderCircleA/CollideeCircleB, ColliderCircleB/CollideeCircleA
				//points = append(points, collideCirclesCircles(colliderCenterA, colliderRadiusA, collideeCenterA, collideeRadiusA, colliderCenterB, colliderRadiusB, collideeCenterB, collideeRadiusB)...)
			}
		}

	}

}
