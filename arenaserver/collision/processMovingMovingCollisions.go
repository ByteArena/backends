package collision

import (
	"sync"

	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
)

func ProcessMovingMovingCollisions(movements []*MovementState, rtMoving *rtreego.Rtree) []Collision {

	//show := spew.ConfigState{MaxDepth: 5, Indent: "    "}

	collisions := make([]Collision, 0)
	collisionsMutex := &sync.Mutex{}

	wait := &sync.WaitGroup{}
	wait.Add(len(movements))

	memoizedCollisions := NewMemoizedMovingMovingCollisions()

	for _, movement := range movements {
		func(movement *MovementState) {

			matchingObjects := rtMoving.SearchIntersect(movement.Rect, func(results []rtreego.Spatial, object rtreego.Spatial) (refuse, abort bool) {
				return object == movement, false // avoid object overlapping itself
			})

			if len(matchingObjects) == 0 {
				wait.Done()
				return
			}

			fineCollisionMovingMoving(movement, matchingObjects, nil, func(colls []Collision) {
				collisionsMutex.Lock()
				collisions = append(collisions, colls...)
				collisionsMutex.Unlock()
			}, memoizedCollisions)

			wait.Done()
		}(movement)
	}

	wait.Wait()

	return collisions
}

func fineCollisionMovingMoving(movement *MovementState, matchingObstacles []rtreego.Spatial, geotypesIgnored []int, collisionhandler func(colls []Collision), memoizedCollisions *memoizedMovingMovingCollisions) {

	//show := spew.ConfigState{MaxDepth: 5, Indent: "    "}

	// Il faut vérifier l'intersection des (polygones + end circles) de trajectoire de Collider et de Collidee

	// On détermine les end circles de la trajectoire du collider
	colliderCenterA := movement.Before.Position
	colliderRadiusA := movement.Before.Radius

	colliderCenterB := movement.After.Position
	colliderRadiusB := movement.After.Radius

	colliderPoly := makePoly(colliderCenterA, colliderCenterB, colliderRadiusA, colliderRadiusB)

	for _, matchingObstacle := range matchingObstacles {

		geoObject := matchingObstacle.(*MovementState)

		if movement.Type == state.GeometryObjectType.Agent && geoObject.GetType() == state.GeometryObjectType.Projectile && geoObject.AgentEmitterID == movement.ID {
			// Avoiding self-collisions early to save processing power, and because it somehow allows agent to penetrate the geometry if not
			continue
		}

		if movement.Type == state.GeometryObjectType.Projectile && geoObject.GetType() == state.GeometryObjectType.Agent && movement.AgentEmitterID == geoObject.ID {
			// Avoiding self-collisions early to save processing power, and because it somehow allows agent to penetrate the geometry if not
			continue
		}

		if movement.Type == state.GeometryObjectType.Projectile && geoObject.GetType() == state.GeometryObjectType.Projectile && movement.AgentEmitterID == geoObject.AgentEmitterID {
			// Avoiding self-collisions early to save processing power, and because it somehow allows agent to penetrate the geometry if not
			continue
		}

		// memoized := memoizedCollisions.get(
		// 	movement.Type,
		// 	movement.ID,
		// 	geoObject.GetType(),
		// 	geoObject.GetID(),
		// )
		// if memoized != nil {
		// 	//log.Println("FROM MEMOIZATION !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
		// 	collisionhandler([]Collision{*memoized})
		// 	continue
		// }

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
						colls := []Collision{{
							ColliderType:      movement.Type,
							ColliderID:        movement.ID,
							CollideeType:      other.GetType(),
							CollideeID:        other.GetID(),
							Point:             vector.MakeSegment2(firstColliderPositionWhenColliding, lastColliderPositionWhenColliding).Center(),
							ColliderTimeBegin: tBeginCollider,
							ColliderTimeEnd:   tEndCollider,
							ColliderMovement:  movement,
						}}
						//memoizedCollisions.add(colls)
						collisionhandler(colls)
					}
				}

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
					colls := []Collision{{
						ColliderType:      movement.Type,
						ColliderID:        movement.ID,
						CollideeType:      other.GetType(),
						CollideeID:        other.GetID(),
						Point:             vector.MakeSegment2(firstColliderPositionWhenColliding, lastColliderPositionWhenColliding).Center(),
						ColliderTimeBegin: tBeginCollider,
						ColliderTimeEnd:   tEndCollider,
						ColliderMovement:  movement,
					}}
					if movement.Type == 3 && other.GetType() == 2 {
						//show.Dump("PROJECTILE INCOMING ON AGENT !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", colls)
					}
					//they were at the same place at the same time in the tick ! Collision !
					//memoizedCollisions.add(colls)
					collisionhandler(colls)
				}

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
						colls := []Collision{{
							ColliderType:      movement.Type,
							ColliderID:        movement.ID,
							CollideeType:      other.GetType(),
							CollideeID:        other.GetID(),
							Point:             vector.MakeSegment2(firstColliderPositionWhenColliding, lastColliderPositionWhenColliding).Center(),
							ColliderTimeBegin: tBeginCollider,
							ColliderTimeEnd:   tEndCollider,
							ColliderMovement:  movement,
						}}
						//memoizedCollisions.add(colls)
						collisionhandler(colls)
					}
				}

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
