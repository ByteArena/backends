package game

import (
	"math"

	"github.com/bytearena/box2d"
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/ecs"
)

func (game *DeathmatchGame) ComputeAgentVision(arenaMap *mapcontainer.MapContainer, entity *ecs.Entity, physicalAspect *PhysicalBody, perceptionAspect *Perception) []protocol.AgentPerceptionVisionItem {

	vision := make([]protocol.AgentPerceptionVisionItem, 0)

	// Vision: Les autres agents
	vision = append(vision, viewAgents(game, entity, physicalAspect, perceptionAspect)...)

	// Vision: les obstacles
	//vision = append(vision, viewObstacles(game, entity)...)

	return vision
}

func viewAgents(game *DeathmatchGame, entity *ecs.Entity, physicalAspect *PhysicalBody, perceptionAspect *Perception) []protocol.AgentPerceptionVisionItem {

	vision := make([]protocol.AgentPerceptionVisionItem, 0)

	agentposition := physicalAspect.GetPosition()

	orientation := physicalAspect.GetOrientation()
	radiussq := math.Pow(perceptionAspect.GetVisionRadius(), 2)

	for _, otherentityresult := range game.agentsView.Get() {

		otherentity := otherentityresult.Entity

		if otherentity.GetID() == entity.GetID() {
			continue // one cannot see itself
		}

		otherPhysicalAspect := game.CastPhysicalBody(otherentityresult.Components[game.physicalBodyComponent.GetID()])

		otherPosition := otherPhysicalAspect.GetPosition()
		otherVelocity := otherPhysicalAspect.GetVelocity()
		otherRadius := otherPhysicalAspect.GetRadius()

		centervec := otherPosition.Sub(agentposition)
		centersegment := vector.MakeSegment2(vector.MakeNullVector2(), centervec)
		agentdiameter := centersegment.OrthogonalToBCentered().SetLengthFromCenter(otherRadius * 2)

		closeEdge, farEdge := agentdiameter.Get()

		distsq := centervec.MagSq()
		if distsq <= radiussq {

			occulted := false

			// raycast between two the agents to determine if they can see each other
			game.PhysicalWorld.RayCast(
				func(fixture *box2d.B2Fixture, point box2d.B2Vec2, normal box2d.B2Vec2, fraction float64) float64 {
					bodyDescriptor, ok := fixture.GetBody().GetUserData().(types.PhysicalBodyDescriptor)
					if !ok {
						return 1.0 // continue the ray
					}

					if bodyDescriptor.Type == types.PhysicalBodyDescriptorType.Obstacle {
						occulted = true
						return 0.0 // terminate the ray
					}

					return 1.0 // continue the ray
				},
				agentposition.ToB2Vec2(),
				otherPosition.ToB2Vec2(),
			)

			if occulted {
				continue // cannot see through obstacles
			}

			// Il faut aligner l'angle du vecteur sur le heading courant de l'agent
			centervec = centervec.SetAngle(centervec.Angle() - orientation)
			visionitem := protocol.AgentPerceptionVisionItem{
				CloseEdge: closeEdge.Clone().SetAngle(closeEdge.Angle() - orientation), // perpendicular to relative position vector, left side
				Center:    centervec,
				FarEdge:   farEdge.Clone().SetAngle(farEdge.Angle() - orientation), // perpendicular to relative position vector, right side
				// FIXME(jerome): /20 here is to convert velocity per second in velocity per tick; should probably handle velocities in m/s everywhere ?
				Velocity: otherVelocity.Clone().SetAngle(otherVelocity.Angle() - orientation).Scale(1 / 20),
				Tag:      protocol.AgentPerceptionVisionItemTag.Agent,
			}

			vision = append(vision, visionitem)

			//log.Println(orientation, otherVelocity, closeEdge, farEdge, visionitem)
		}
	}

	return vision
}

func viewObstacles(game *DeathmatchGame, entity *ecs.Entity) []protocol.AgentPerceptionVisionItem {

	vision := make([]protocol.AgentPerceptionVisionItem, 0)

	// FIXME(jerome)
	// physicalAspect := game.GetPhysicalBody(entity)
	// perceptionAspect := game.GetPerception(entity)

	// absoluteposition := physicalAspect.GetPosition()
	// orientation := physicalAspect.GetOrientation()
	// visionradius := perceptionAspect.GetVisionRadius()
	// visionangle := perceptionAspect.GetVisionAngle()

	// radiussq := math.Pow(perceptionAspect.GetVisionRadius(), 2)

	// // On détermine les bords gauche et droit du cône de vision de l'agent
	// halfvisionangle := visionangle / 2
	// leftvisionrelvec := vector.MakeVector2(1, 1).SetMag(visionradius).SetAngle(orientation + halfvisionangle*-1)
	// rightvisionrelvec := vector.MakeVector2(1, 1).SetMag(visionradius).SetAngle(orientation + halfvisionangle)

	// for _, obstacle := range game.MapMemoization.Obstacles {

	// 	edges := make([]vector.Vector2, 0)
	// 	//rejectededges := make([]vector.Vector2, 0)

	// 	relvecA := obstacle.A.Sub(absoluteposition)
	// 	relvecB := obstacle.B.Sub(absoluteposition)

	// 	distsqA := relvecA.MagSq()
	// 	distsqB := relvecB.MagSq()

	// 	// Comment déterminer si le vecteur entre dans le champ de vision ?
	// 	// => Intersection entre vecteur et segment gauche, droite

	// 	if distsqA <= radiussq {
	// 		// in radius
	// 		absAngleA := relvecA.Angle()
	// 		relAngleA := absAngleA - orientation

	// 		// On passe de 0° / 360° à -180° / +180°
	// 		relAngleA = trigo.FullCircleAngleToSignedHalfCircleAngle(relAngleA)

	// 		if math.Abs(relAngleA) <= halfvisionangle {
	// 			// point dans le champ de vision !
	// 			edges = append(edges, relvecA.Add(absoluteposition))
	// 		} else {
	// 			//rejectededges = append(rejectededges, relvecA.Add(absoluteposition))
	// 		}
	// 	}

	// 	if distsqB <= radiussq {
	// 		absAngleB := relvecB.Angle()
	// 		relAngleB := absAngleB - orientation

	// 		// On passe de 0° / 360° à -180° / +180°
	// 		relAngleB = trigo.FullCircleAngleToSignedHalfCircleAngle(relAngleB)

	// 		if math.Abs(relAngleB) <= halfvisionangle {
	// 			// point dans le champ de vision !
	// 			edges = append(edges, relvecB.Add(absoluteposition))
	// 		} else {
	// 			//rejectededges = append(rejectededges, relvecB.Add(absoluteposition))
	// 		}
	// 	}

	// 	{
	// 		// Sur les bords de la perception
	// 		if point, intersects, colinear, _ := trigo.IntersectionWithLineSegment(vector.MakeNullVector2(), leftvisionrelvec, relvecA, relvecB); intersects && !colinear {
	// 			// INTERSECT LEFT
	// 			edges = append(edges, point.Add(absoluteposition))
	// 		}

	// 		if point, intersects, colinear, _ := trigo.IntersectionWithLineSegment(vector.MakeNullVector2(), rightvisionrelvec, relvecA, relvecB); intersects && !colinear {
	// 			// INTERSECT RIGHT
	// 			edges = append(edges, point.Add(absoluteposition))
	// 		}
	// 	}

	// 	{
	// 		// Sur l'horizon de perception (arc de cercle)
	// 		intersections := trigo.LineCircleIntersectionPoints(
	// 			relvecA,
	// 			relvecB,
	// 			vector.MakeNullVector2(),
	// 			agentstate.VisionRadius,
	// 		)

	// 		for _, point := range intersections {
	// 			// il faut vérifier que le point se trouve bien sur le segment
	// 			// il faut vérifier que l'angle du point de collision se trouve bien dans le champ de vision de l'agent

	// 			if trigo.PointOnLineSegment(point, relvecA, relvecB) {
	// 				relvecangle := point.Angle() - orientation

	// 				// On passe de 0° / 360° à -180° / +180°
	// 				relvecangle = trigo.FullCircleAngleToSignedHalfCircleAngle(relvecangle)

	// 				if math.Abs(relvecangle) <= halfvisionangle {
	// 					edges = append(edges, point.Add(absoluteposition))
	// 				} else {
	// 					//rejectededges = append(rejectededges, point.Add(absoluteposition))
	// 				}
	// 			} else {
	// 				//rejectededges = append(rejectededges, point.Add(absoluteposition))
	// 			}
	// 		}
	// 	}

	// 	if len(edges) == 2 {
	// 		edgeone := edges[0]
	// 		edgetwo := edges[1]
	// 		center := edgetwo.Add(edgeone).DivScalar(2)

	// 		//visiblemag := edgetwo.Sub(edgeone).Mag()

	// 		relcenter := center.Sub(absoluteposition) // aligned on north
	// 		relcenterangle := relcenter.Angle()
	// 		relcenteragentaligned := relcenter.SetAngle(relcenterangle - orientation)

	// 		relEdgeOne := edgeone.Sub(absoluteposition)
	// 		relEdgeTwo := edgetwo.Sub(absoluteposition)

	// 		relEdgeOneAgentAligned := relEdgeOne.SetAngle(relEdgeOne.Angle() - orientation)
	// 		relEdgeTwoAgentAligned := relEdgeTwo.SetAngle(relEdgeTwo.Angle() - orientation)

	// 		var closeEdge, farEdge vector.Vector2
	// 		if relEdgeTwoAgentAligned.MagSq() > relEdgeOneAgentAligned.MagSq() {
	// 			closeEdge = relEdgeOneAgentAligned
	// 			farEdge = relEdgeTwoAgentAligned
	// 		} else {
	// 			closeEdge = relEdgeTwoAgentAligned
	// 			farEdge = relEdgeOneAgentAligned
	// 		}

	// 		obstacleperception := protocol.AgentPerceptionVisionItem{
	// 			CloseEdge: closeEdge,
	// 			Center:    relcenteragentaligned,
	// 			FarEdge:   farEdge,
	// 			Velocity:  vector.MakeNullVector2(),
	// 			Tag:       protocol.AgentPerceptionVisionItemTag.Obstacle,
	// 		}

	// 		vision = append(vision, obstacleperception)

	// 	} else if len(edges) > 0 {
	// 		// problems with FOV > 180
	// 	}
	// }

	return vision
}
