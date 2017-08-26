package arenaserver

import (
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/dhconnelly/rtreego"
	uuid "github.com/satori/go.uuid"
)

func updateProjectiles(server *Server) (beforeStates map[uuid.UUID]movingObjectTemporaryState) {

	server.state.Projectilesmutex.Lock()

	projectilesToRemove := make([]uuid.UUID, 0)
	for _, projectile := range server.state.Projectiles {
		if projectile.TTL <= 0 {
			projectilesToRemove = append(projectilesToRemove, projectile.Id)
		}
	}

	for _, projectileToRemoveId := range projectilesToRemove {
		// has been set to 0 during the previous tick; pruning now (0 TTL projectiles might still have a collision later in this method)
		// Remove projectile from projectiles array
		delete(server.state.Projectiles, projectileToRemoveId)
	}

	before := make(map[uuid.UUID]movingObjectTemporaryState)
	for _, projectile := range server.state.Projectiles {
		before[projectile.Id] = movingObjectTemporaryState{
			Position: projectile.Position,
			Velocity: projectile.Velocity,
			Radius:   projectile.Radius,
		}
	}

	for _, projectile := range server.state.Projectiles {
		projectile.Update()
	}

	server.state.Projectilesmutex.Unlock()

	return before
}

func updateAgents(server *Server) (beforeStates map[uuid.UUID]movingObjectTemporaryState) {

	before := make(map[uuid.UUID]movingObjectTemporaryState)

	for _, agent := range server.agents {
		id := agent.GetId()
		agstate := server.state.GetAgentState(id)
		before[id] = movingObjectTemporaryState{
			Position: agstate.Position,
			Velocity: agstate.Velocity,
			Radius:   agstate.Radius,
		}
	}

	for _, agent := range server.agents {
		server.state.SetAgentState(
			agent.GetId(),
			server.state.GetAgentState(agent.GetId()).Update(),
		)
	}

	return before
}

func processProjectileObstacleCollisions(server *Server, before map[uuid.UUID]movingObjectTemporaryState) {
	for projectileid, beforestate := range before {
		projectile := server.state.GetProjectile(projectileid)

		afterstate := movingObjectTemporaryState{
			Position: projectile.Position,
			Velocity: projectile.Velocity,
			Radius:   projectile.Radius,
		}

		processMovingObjectObstacleCollision(server, beforestate, afterstate, []int{state.GeometryObjectType.ObstacleGround}, func(collisionPoint vector.Vector2) {

			projectile.Position = collisionPoint
			projectile.Velocity = vector.MakeNullVector2()
			server.state.SetProjectile(
				projectileid,
				projectile,
			)
		})
	}
}

func processAgentObstacleCollisions(server *Server, before map[uuid.UUID]movingObjectTemporaryState) {

	for agentid, beforestate := range before {
		agentstate := server.state.GetAgentState(agentid)

		afterstate := movingObjectTemporaryState{
			Position: agentstate.Position,
			Velocity: agentstate.Velocity,
			Radius:   agentstate.Radius,
		}

		processMovingObjectObstacleCollision(server, beforestate, afterstate, nil, func(collisionPoint vector.Vector2) {
			agentstate.Position = collisionPoint
			agentstate.Velocity = vector.MakeVector2(0.01, 0.01)
			server.state.SetAgentState(
				agentid,
				agentstate,
			)
		})
	}
}

func arrayContainsGeotype(needle int, haystack []int) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func processMovingObjectObstacleCollision(server *Server, beforeState, afterState movingObjectTemporaryState, geotypesIgnored []int, collisionhandler func(collision vector.Vector2)) {

	bbBeforeA, bbBeforeB := GetAgentBoundingBox(beforeState.Position, beforeState.Radius)
	bbAfterA, bbAfterB := GetAgentBoundingBox(afterState.Position, afterState.Radius)

	var minX, minY *float64
	var maxX, maxY *float64

	for _, point := range []vector.Vector2{bbBeforeA, bbBeforeB, bbAfterA, bbAfterB} {

		x, y := point.Get()

		if minX == nil || x < *minX {
			minX = &(x)
		}

		if minY == nil || y < *minY {
			minY = &(y)
		}

		if maxX == nil || x > *maxX {
			maxX = &(x)
		}

		if maxY == nil || y > *maxY {
			maxY = &(y)
		}
	}

	bbRegion, err := rtreego.NewRect([]float64{*minX, *minY}, []float64{*maxX - *minX, *maxY - *minY})
	if err != nil {
		utils.Debug("arena-server-updatestate", "Error in processMovingObjectObstacleCollision: vould not define bbRegion in obstacle rTree")
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

		type Collision struct {
			Point    vector.Vector2
			Obstacle *state.GeometryObject
		}

		collisions := make([]Collision, 0)

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
				collisions = append(collisions, Collision{
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
					collisions = append(collisions, Collision{
						Point:    collisionPoint,
						Obstacle: geoObject,
					})
				}
			}
		}

		if len(collisions) > 0 {

			//normal := vector.MakeNullVector2()
			minDist := -1.0
			for _, collision := range collisions {
				thisDist := collision.Point.Sub(beforeState.Position).Mag()
				if minDist < 0 || minDist > thisDist {
					minDist = thisDist
					//normal = collision.Obstacle.Normal
				}
			}

			//normal = normal.Normalize()

			backoffDistance := beforeState.Radius + 0.001
			//nextPoint := centerEdge.Vector2().SetMag(maxDist).Sub(normal.SetMag(backoffDistance)).Add(beforeState.Position)
			nextPoint := centerEdge.Vector2().SetMag(minDist - backoffDistance).Add(beforeState.Position)

			if !isInsideGroundSurface(server, nextPoint) {

				// backtracking position to last not outside
				backsteps := 10
				railRel := afterState.Position.Sub(beforeState.Position)
				for k := 1; k <= backsteps; k++ {
					nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
					if isInsideGroundSurface(server, nextPointRel.Add(beforeState.Position)) {
						collisionhandler(nextPointRel.Add(beforeState.Position))
						return
					}
				}

				collisionhandler(beforeState.Position)

			} else {
				if isInsideCollisionMesh(server, nextPoint) {
					if isInsideCollisionMesh(server, beforeState.Position) {
						// moving it outside the mesh !!
						railRel := afterState.Position.Sub(beforeState.Position)
						railRel = railRel.Sub(railRel.SetMag(0.1))
						collisionhandler(railRel.Add(beforeState.Position))

					} else {
						// backtracking position to last not in obstacle
						backsteps := 30
						railRel := afterState.Position.Sub(beforeState.Position)
						railRel = railRel.Sub(railRel.SetMag(0.05))
						for k := 1; k <= backsteps; k++ {
							nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
							if !isInsideCollisionMesh(server, nextPointRel.Add(beforeState.Position)) {
								collisionhandler(nextPointRel.Add(beforeState.Position))
								return
							}
						}

						collisionhandler(beforeState.Position)
					}

				} else {
					collisionhandler(nextPoint)
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

func GetAgentBoundingBox(center vector.Vector2, radius float64) (vector.Vector2, vector.Vector2) {
	x, y := center.Get()
	return vector.MakeVector2(x-radius, y-radius), vector.MakeVector2(x+radius, y+radius)
}
