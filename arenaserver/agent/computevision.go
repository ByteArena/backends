package agent

import (
	"log"
	"math"

	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
)

func (agent AgentImp) computeAgentVision(serverstate *state.ServerState, agentstate state.AgentState) []state.PerceptionVisionItem {

	vision := make([]state.PerceptionVisionItem, 0)

	absoluteposition := agentstate.Position
	orientation := agentstate.Orientation

	// On calcule la perception Vision de l'agent
	serverstate.Agentsmutex.Lock()
	radiussq := agentstate.VisionRadius * agentstate.VisionRadius
	for otheragentid, otheragentstate := range serverstate.Agents {

		if otheragentid == agent.GetId() {
			continue
		}

		centervec := otheragentstate.Position.Sub(agentstate.Position)
		distsq := centervec.MagSq()
		if distsq <= radiussq {
			// Il faut aligner l'angle du vecteur sur le heading courant de l'agent
			centervec = centervec.SetAngle(centervec.Angle() - orientation)
			visionitem := state.PerceptionVisionItem{
				CloseEdge: vector.MakeNullVector2(), // TODO: compute actual value for this
				Center:    centervec,
				FarEdge:   vector.MakeNullVector2(), // TODO: compute actual value for this
				Velocity:  otheragentstate.Velocity.Clone().SetAngle(otheragentstate.Velocity.Angle() - orientation),
				Tag:       otheragentstate.Tag,
			}

			vision = append(vision, visionitem)
		}
	}
	serverstate.Agentsmutex.Unlock()

	// Vision: les obstacles
	halfvisionangle := agentstate.VisionAngle / 2
	leftvisionrelvec := vector.MakeVector2(1, 1).SetMag(agentstate.VisionRadius).SetAngle(orientation + halfvisionangle*-1)
	rightvisionrelvec := vector.MakeVector2(1, 1).SetMag(agentstate.VisionRadius).SetAngle(orientation + halfvisionangle)

	serverstate.Obstaclesmutex.Lock()
	for _, obstacle := range serverstate.Obstacles {

		edges := make([]vector.Vector2, 0)
		rejectededges := make([]vector.Vector2, 0)

		relvecA := obstacle.A.Sub(absoluteposition)
		relvecB := obstacle.B.Sub(absoluteposition)

		distsqA := relvecA.MagSq()
		distsqB := relvecB.MagSq()

		// Comment déterminer si le vecteur entre dans le champ de vision ?
		// => Intersection entre vecteur et segment gauche, droite

		if distsqA <= radiussq {
			// in radius
			absAngleA := relvecA.Angle()
			relAngleA := absAngleA - orientation

			// On passe de 0° / 360° à -180° / +180°
			relAngleA = trigo.FullCircleAngleToSignedHalfCircleAngle(relAngleA)

			if math.Abs(relAngleA) <= halfvisionangle {
				// point dans le champ de vision !
				edges = append(edges, relvecA.Add(absoluteposition))
			} else {
				//rejectededges = append(rejectededges, relvecA.Add(absoluteposition))
			}
		}

		if distsqB <= radiussq {
			absAngleB := relvecB.Angle()
			relAngleB := absAngleB - orientation

			// On passe de 0° / 360° à -180° / +180°
			relAngleB = trigo.FullCircleAngleToSignedHalfCircleAngle(relAngleB)

			if math.Abs(relAngleB) <= halfvisionangle {
				// point dans le champ de vision !
				edges = append(edges, relvecB.Add(absoluteposition))
			} else {
				//rejectededges = append(rejectededges, relvecB.Add(absoluteposition))
			}
		}

		{
			// Sur les bords de la perception
			if point, intersects, colinear, _ := trigo.IntersectionWithLineSegment(vector.MakeNullVector2(), leftvisionrelvec, relvecA, relvecB); intersects && !colinear {
				// INTERSECT LEFT
				edges = append(edges, point.Add(absoluteposition))
			}

			if point, intersects, colinear, _ := trigo.IntersectionWithLineSegment(vector.MakeNullVector2(), rightvisionrelvec, relvecA, relvecB); intersects && !colinear {
				// INTERSECT RIGHT
				edges = append(edges, point.Add(absoluteposition))
			}
		}

		{
			// Sur l'horizon de perception (arc de cercle)
			intersections := trigo.LineCircleIntersectionPoints(
				relvecA,
				relvecB,
				vector.MakeNullVector2(),
				agentstate.VisionRadius,
			)

			for _, point := range intersections {
				// il faut vérifier que le point se trouve bien sur le segment
				// il faut vérifier que l'angle du point de collision se trouve bien dans le champ de vision de l'agent

				if trigo.PointOnLineSegment(point, relvecA, relvecB) {
					relvecangle := point.Angle() - orientation

					// On passe de 0° / 360° à -180° / +180°
					relvecangle = trigo.FullCircleAngleToSignedHalfCircleAngle(relvecangle)

					if math.Abs(relvecangle) <= halfvisionangle {
						edges = append(edges, point.Add(absoluteposition))
					} else {
						//rejectededges = append(rejectededges, point.Add(absoluteposition))
					}
				} else {
					//rejectededges = append(rejectededges, point.Add(absoluteposition))
				}
			}
		}

		if len(edges) == 2 {
			edgeone := edges[0]
			edgetwo := edges[1]
			center := edgetwo.Add(edgeone).DivScalar(2)

			//visiblemag := edgetwo.Sub(edgeone).Mag()

			relcenter := center.Sub(absoluteposition) // aligned on north
			relcenterangle := relcenter.Angle()
			relcenteragentaligned := relcenter.SetAngle(relcenterangle - orientation)

			relEdgeOne := edgeone.Sub(absoluteposition)
			relEdgeTwo := edgetwo.Sub(absoluteposition)

			relEdgeOneAgentAligned := relEdgeOne.SetAngle(relEdgeOne.Angle() - orientation)
			relEdgeTwoAgentAligned := relEdgeTwo.SetAngle(relEdgeTwo.Angle() - orientation)

			var closeEdge, farEdge vector.Vector2
			if relEdgeTwoAgentAligned.MagSq() > relEdgeOneAgentAligned.MagSq() {
				closeEdge = relEdgeOneAgentAligned
				farEdge = relEdgeTwoAgentAligned
			} else {
				closeEdge = relEdgeTwoAgentAligned
				farEdge = relEdgeOneAgentAligned
			}

			obstacleperception := state.PerceptionVisionItem{
				CloseEdge: closeEdge,
				Center:    relcenteragentaligned,
				FarEdge:   farEdge,
				Velocity:  vector.MakeNullVector2(),
				Tag:       "obstacle",
			}

			vision = append(vision, obstacleperception)

		} else if len(edges) > 0 {
			log.Println("NOPE !", edges)
		}

		for _, edge := range edges {
			serverstate.DebugIntersects = append(serverstate.DebugIntersects, edge)
		}

		for _, edge := range rejectededges {
			serverstate.DebugIntersectsRejected = append(serverstate.DebugIntersectsRejected, edge)
		}
	}
	serverstate.Obstaclesmutex.Unlock()

	return vision
}
