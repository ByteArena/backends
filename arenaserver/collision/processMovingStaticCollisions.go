package collision

import (
	"sync"

	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
	uuid "github.com/satori/go.uuid"
)

func ProcessMovingStaticCollisions(before map[uuid.UUID]CollisionMovingObjectState, mapMemoization *state.MapMemoization, colliderType int, geoTypesIgnored []int, getMovingObjectState func(objectid uuid.UUID) CollisionMovingObjectState) []Collision {

	collisions := make([]Collision, 0)
	collisionsMutex := &sync.Mutex{}

	wait := &sync.WaitGroup{}
	wait.Add(len(before))

	for objectid, beforestate := range before {
		go func(beforestate CollisionMovingObjectState, objectid uuid.UUID) {

			afterstate := getMovingObjectState(objectid)

			bbRegion, err := GetTrajectoryBoundingBox(beforestate.Position, beforestate.Radius, afterstate.Position, afterstate.Radius)
			if err != nil {
				utils.Debug("arena-server-updatestate", "Error in processMovingObjectObstacleCollision: could not define bbRegion in obstacle rTree")
				return
			}

			matchingObstacles := mapMemoization.RtreeObstacles.SearchIntersect(bbRegion)

			if len(matchingObstacles) > 0 {
				// fine collision checking
				fineCollisionChecking(mapMemoization, beforestate, afterstate, matchingObstacles, geoTypesIgnored, func(collisionPoint vector.Vector2, other FinelyCollisionable) {

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
					collisions = append(collisions, Collision{
						ColliderType:      colliderType,
						ColliderID:        objectid.String(),
						CollideeType:      other.GetType(),
						CollideeID:        other.GetID(),
						Point:             collisionPoint,
						ColliderTimeBegin: tBegin,
						ColliderTimeEnd:   tEnd,
					})
					collisionsMutex.Unlock()
				})
			}

			wait.Done()

		}(beforestate, objectid)
	}

	wait.Wait()

	return collisions
}

func arrayContainsGeotype(needle int, haystack []int) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func fineCollisionChecking(mapMemoization *state.MapMemoization, beforeState, afterState CollisionMovingObjectState, matchingObstacles []rtreego.Spatial, geotypesIgnored []int, collisionhandler collisionHandlerFunc) {
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
		geoObject := matchingObstacle.(FinelyCollisionable)
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

		if isInsideGroundSurface(mapMemoization, nextPoint) {
			if isInsideCollisionMesh(mapMemoization, nextPoint) {
				if isInsideCollisionMesh(mapMemoization, beforeState.Position) {
					// moving it outside the mesh !!
					railRel := afterState.Position.Sub(beforeState.Position)
					railRel = railRel.Sub(railRel.SetMag(0.1))
					//log.Println("TWOOOOOOOOOOOOOOOOOOOOO", firstObstacle.GetType())
					collisionhandler(railRel.Add(beforeState.Position), firstCollision.Obstacle)

				} else {
					// backtracking position to last not in obstacle
					backsteps := 10
					railRel := nextPoint.Sub(beforeState.Position)
					railRel = railRel.Sub(railRel.SetMag(0.05))
					for k := 1; k <= backsteps; k++ {
						nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
						if !isInsideCollisionMesh(mapMemoization, nextPointRel.Add(beforeState.Position)) {
							//log.Println("THREEEEEEEEEEEEEEEEEEEEEEEE", firstObstacle.GetType())
							collisionhandler(nextPointRel.Add(beforeState.Position), firstCollision.Obstacle)
							return
						}
					}

					//log.Println("FOUUUUUUUUUUUUUUUR", firstObstacle.GetType())
					collisionhandler(beforeState.Position, firstCollision.Obstacle)
				}

			} else {

				if firstCollision.Obstacle.GetType() == state.GeometryObjectType.ObstacleObject {
					backsteps := 10
					railRel := nextPoint.Sub(beforeState.Position)
					railRel = railRel.Sub(railRel.SetMag(0.05))
					for k := 0; k <= backsteps; k++ {
						nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
						if !isInsideCollisionMesh(mapMemoization, nextPointRel.Add(beforeState.Position)) {
							//log.Println("FIIIIIIIIIIIVE", firstObstacle.GetType())
							collisionhandler(nextPointRel.Add(beforeState.Position), firstCollision.Obstacle)
							return
						}
					}
				}

				//log.Println("SIIIIIIIIIIIIIIIIIIIIIX", firstObstacle.GetType())
				collisionhandler(nextPoint, firstCollision.Obstacle)
			}

		} else {

			// backtracking position to last not outside
			backsteps := 10
			railRel := nextPoint.Sub(beforeState.Position)
			for k := 1; k <= backsteps; k++ {
				nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
				if isInsideGroundSurface(mapMemoization, nextPointRel.Add(beforeState.Position)) {
					//log.Println("ZEEEEEEEEEEEEEEEROOOOOOOOOOOOO", firstObstacle.GetType())
					collisionhandler(nextPointRel.Add(beforeState.Position), firstCollision.Obstacle)
					return
				}
			}

			//log.Println("OOOOOOOOOOOOOOOOOOONE", firstObstacle.GetType())
			collisionhandler(beforeState.Position, firstCollision.Obstacle)
		}

	}
}
