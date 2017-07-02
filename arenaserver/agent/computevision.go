package agent

import (
	"log"
	"math"

	agstate "github.com/bytearena/bytearena/arenaserver/state/agent"
	stateprotocol "github.com/bytearena/bytearena/arenaserver/state/protocol"
	srvstate "github.com/bytearena/bytearena/arenaserver/state/server"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
)

func (agent AgentImp) computeAgentVision(serverstate *srvstate.ServerState, agentstate stateprotocol.AgentStateInterface) []agstate.PerceptionVisionItem {

	vision := make([]agstate.PerceptionVisionItem, 0)

	orientation := agentstate.GetOrientation()
	absoluteposition := agentstate.GetPosition()
	//velocity := agentstate.GetVelocity()
	//radius := agentstate.GetRadius()
	visionRadius := agentstate.GetVisionRadius()
	visionAngle := agentstate.GetVisionAngle()

	// On détermine les bords gauche et droit du cône de vision de l'agent
	halfvisionangle := visionAngle / 2
	leftvisionrelvec := vector.MakeVector2(1, 1).SetMag(visionRadius).SetAngle(orientation + halfvisionangle*-1)
	rightvisionrelvec := vector.MakeVector2(1, 1).SetMag(visionRadius).SetAngle(orientation + halfvisionangle)

	// On détermine la perception visuelle de l'agent

	///////////////////////////////////////////////////////////////////////////
	// Vision: Les autres agents
	///////////////////////////////////////////////////////////////////////////

	serverstate.Agentsmutex.Lock()
	radiussq := visionRadius * visionRadius
	for otheragentid, otheragentstate := range serverstate.Agents {

		if otheragentid == agent.GetId() {
			continue // one cannot see itself
		}

		centervec := otheragentstate.GetPosition().Sub(absoluteposition)
		centersegment := vector.MakeSegment2(vector.MakeNullVector2(), centervec)
		agentdiameter := centersegment.OrthogonalToBCentered().SetLengthFromCenter(otheragentstate.GetRadius() * 2)

		closeEdge, farEdge := agentdiameter.Get()

		distsq := centervec.MagSq()
		if distsq <= radiussq {
			// Il faut aligner l'angle du vecteur sur le heading courant de l'agent
			centervec = centervec.SetAngle(centervec.Angle() - orientation)
			visionitem := agstate.PerceptionVisionItem{
				CloseEdge: closeEdge.Clone().SetAngle(closeEdge.Angle() - orientation), // perpendicular to relative position vector, left side
				Center:    centervec,
				FarEdge:   farEdge.Clone().SetAngle(farEdge.Angle() - orientation), // perpendicular to relative position vector, right side
				Velocity:  otheragentstate.GetVelocity().Clone().SetAngle(otheragentstate.GetVelocity().Angle() - orientation),
				//Tag:       otheragentstate.Tag,
				Tag: "agent",
			}

			vision = append(vision, visionitem)
		}
	}
	serverstate.Agentsmutex.Unlock()

	///////////////////////////////////////////////////////////////////////////
	// Vision: les obstacles
	///////////////////////////////////////////////////////////////////////////

	serverstate.Obstaclesmutex.Lock()
	for _, obstacle := range serverstate.Obstacles {

		edges := make([]vector.Vector2, 0)
		rejectededges := make([]vector.Vector2, 0)

		relvecA := obstacle.GetA().Sub(absoluteposition)
		relvecB := obstacle.GetB().Sub(absoluteposition)

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
				visionRadius,
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

			obstacleperception := agstate.PerceptionVisionItem{
				CloseEdge: closeEdge,
				Center:    relcenteragentaligned,
				FarEdge:   farEdge,
				Velocity:  vector.MakeNullVector2(),
				Tag:       "obstacle",
			}

			vision = append(vision, obstacleperception)

		} else if len(edges) > 0 {
			log.Println("NOPE !", edges) // problems with FOV > 180
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
