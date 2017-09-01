package collision

import (
	"log"

	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/utils/vector"
)

func EnsureValidPositionAfterCollision(mapMemoization *state.MapMemoization, coll Collision) vector.Vector2 {

	log.Println("ICIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIIII")

	nextPoint := coll.Point
	movement := coll.ColliderMovement

	if isInsideGroundSurface(mapMemoization, nextPoint) {
		log.Println("LAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
		if isInsideCollisionMesh(mapMemoization, nextPoint) {
			return movement.Before.Position
			// log.Println("isInsideCollisionMesh::SAVED---------------------------------------------------------------------------------------------------------------------------------------- !")
			// if isInsideCollisionMesh(mapMemoization, movement.Before.Position) {
			// 	if nextPoint.Equals(movement.Before.Position) {
			// 		panic("nextpoint equals beforepos !")
			// 	}

			// 	seg := vector.MakeSegment2(movement.Before.Position, nextPoint)
			// 	seg = seg.SetLengthFromB(seg.Length() + 0.1)
			// 	for isInsideCollisionMesh(mapMemoization, seg.GetPointA()) {
			// 		seg = seg.SetLengthFromB(seg.Length() + 0.1)
			// 	}

			// 	return seg.GetPointA()
			// }

			// log.Println("isInsideCollisionMesh::SAVED:TWO------------------------------------------------------------------------------------------------------------------------- !")

			// // backtracking position to last not in obstacle
			// backsteps := 10
			// railRel := nextPoint.Sub(movement.Before.Position)
			// railRel = railRel.Sub(railRel.SetMag(0.05))
			// for k := 1; k <= backsteps; k++ {
			// 	nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
			// 	if !isInsideCollisionMesh(mapMemoization, nextPointRel.Add(movement.Before.Position)) {
			// 		//log.Println("THREEEEEEEEEEEEEEEEEEEEEEEE", firstObstacle.GetType())
			// 		return nextPointRel.Add(movement.Before.Position)
			// 	}
			// }

			// //log.Println("FOUUUUUUUUUUUUUUUR", firstObstacle.GetType())
			// return movement.Before.Position

		}

		if coll.CollideeType == state.GeometryObjectType.ObstacleObject {
			backsteps := 10
			railRel := nextPoint.Sub(movement.Before.Position)
			railRel = railRel.Sub(railRel.SetMag(0.05))
			for k := 0; k <= backsteps; k++ {
				nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
				if !isInsideCollisionMesh(mapMemoization, nextPointRel.Add(movement.Before.Position)) {
					//log.Println("FIIIIIIIIIIIVE", firstObstacle.GetType())
					return nextPointRel.Add(movement.Before.Position)
				}
			}
		}

		//log.Println("SIIIIIIIIIIIIIIIIIIIIIX", firstObstacle.GetType())
		return nextPoint

	}

	log.Println("OUTSIDE !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")

	// backtracking position to last not outside
	backsteps := 10
	railRel := nextPoint.Sub(movement.Before.Position)
	for k := 1; k <= backsteps; k++ {
		nextPointRel := railRel.Scale(1 - float64(k)/float64(backsteps))
		if isInsideGroundSurface(mapMemoization, nextPointRel.Add(movement.Before.Position)) {
			//log.Println("ZEEEEEEEEEEEEEEEROOOOOOOOOOOOO", firstObstacle.GetType())
			return nextPointRel.Add(movement.Before.Position)
		}
	}

	//log.Println("OOOOOOOOOOOOOOOOOOONE", firstObstacle.GetType())
	return movement.Before.Position
}
