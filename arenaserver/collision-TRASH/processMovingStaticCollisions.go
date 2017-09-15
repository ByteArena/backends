package collision

import (
	"sync"

	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
)

func ProcessMovingStaticCollisions(movements []*MovementState, mapMemoization *state.MapMemoization, geoTypesIgnored []int) []Collision {

	collisions := make([]Collision, 0)
	collisionsMutex := &sync.Mutex{}

	wait := &sync.WaitGroup{}
	wait.Add(len(movements))

	for _, movement := range movements {
		go func(movement *MovementState) {

			matchingObstacles := mapMemoization.RtreeObstacles.SearchIntersect(movement.Rect)

			if len(matchingObstacles) > 0 {
				// fine collision checking
				fineCollisionChecking(mapMemoization, movement, matchingObstacles, geoTypesIgnored, func(collisionPoint vector.Vector2, collidee FinelyCollisionable) {

					var tBegin float64 = 0.0
					var tEnd float64 = 1.0

					// distance du point de collision à la position initiale
					distTravel := movement.After.Position.Sub(movement.Before.Position).Mag()
					if distTravel > 0 {
						distCollision := collisionPoint.Sub(movement.Before.Position).Mag()
						tBegin = distCollision / distTravel // l'impact avec un obstacle est immédiat, pas de déplacement sur le point de collision
						tEnd = distCollision / distTravel
					}

					collisionsMutex.Lock()
					collisions = append(collisions, Collision{
						ColliderType:      movement.Type,
						ColliderID:        movement.ID,
						CollideeType:      collidee.GetType(),
						CollideeID:        collidee.GetID(),
						Point:             collisionPoint,
						ColliderTimeBegin: tBegin,
						ColliderTimeEnd:   tEnd,
						ColliderMovement:  movement,
					})
					collisionsMutex.Unlock()
				})
			}

			wait.Done()

		}(movement)
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

func fineCollisionChecking(mapMemoization *state.MapMemoization, movement *MovementState, matchingObstacles []rtreego.Spatial, geotypesIgnored []int, collisionhandler collisionHandlerFunc) {
	// Fine collision checking

	// We determine the surface occupied by the object on it's path
	// * Corresponds to a "pill", where the two ends are the bounding circles occupied by the agents (position before the move and position after the move)
	// * And the surface in between is defined the lines between the left and the right tangents of these circles
	//
	// * We then have to test collisions with the end circle
	//

	centerEdge := vector.MakeSegment2(movement.Before.Position, movement.After.Position)
	beforeDiameterSegment := centerEdge.OrthogonalToACentered().SetLengthFromCenter(movement.Before.Radius * 2)
	afterDiameterSegment := centerEdge.OrthogonalToBCentered().SetLengthFromCenter(movement.After.Radius * 2)

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
				movement.After.Position,
				movement.After.Radius,
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
			thisDist := coll.Point.Sub(movement.Before.Position).Mag()
			if minDist < 0 || minDist > thisDist {
				minDist = thisDist
				firstCollision = &coll
			}
		}

		backoffDistance := movement.Before.Radius
		if minDist > backoffDistance {
			collisionhandler(movement.After.Position.Sub(movement.Before.Position).SetMag(minDist-backoffDistance).Add(movement.Before.Position), firstCollision.Obstacle)
		} else {
			collisionhandler(movement.Before.Position, firstCollision.Obstacle)
		}
	}
}
