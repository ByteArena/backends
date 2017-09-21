package perception

import (
	"math"

	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"
	"github.com/bytearena/bytearena/game/entities"
	uuid "github.com/satori/go.uuid"

	"github.com/bytearena/box2d"
)

func ComputeAgentVision(arenaMap *mapcontainer.MapContainer, serverstate *state.ServerState, agent entities.AgentInterface) []protocol.AgentPerceptionVisionItem {

	agentstate := serverstate.GetAgentState(agent.GetId())
	vision := make([]protocol.AgentPerceptionVisionItem, 0)

	// Vision: Les autres agents
	vision = append(vision, viewAgents(serverstate, agentstate, agent.GetId())...)

	// Vision: les obstacles
	vision = append(vision, viewObstacles(serverstate, agentstate)...)

	return vision
}

func viewAgents(serverstate *state.ServerState, agentstate entities.AgentState, agentid uuid.UUID) []protocol.AgentPerceptionVisionItem {

	agentposition := agentstate.GetPosition()

	vision := make([]protocol.AgentPerceptionVisionItem, 0)

	orientation := agentstate.GetOrientation()
	radiussq := agentstate.VisionRadius * agentstate.VisionRadius

	serverstate.Agentsmutex.Lock()
	for otheragentid, otheragentstate := range serverstate.Agents {

		if otheragentid == agentid {
			continue // one cannot see itself
		}

		occulted := false

		// raycast between two the agents to determine if they can see each other
		serverstate.PhysicalWorld.RayCast(
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
			otheragentstate.GetPosition().ToB2Vec2(),
		)

		if occulted {
			continue // cannot see through obstacles
		}

		centervec := otheragentstate.GetPosition().Sub(agentstate.GetPosition())
		centersegment := vector.MakeSegment2(vector.MakeNullVector2(), centervec)
		agentdiameter := centersegment.OrthogonalToBCentered().SetLengthFromCenter(otheragentstate.GetRadius() * 2)

		closeEdge, farEdge := agentdiameter.Get()

		distsq := centervec.MagSq()
		if distsq <= radiussq {
			// Il faut aligner l'angle du vecteur sur le heading courant de l'agent
			centervec = centervec.SetAngle(centervec.Angle() - orientation)
			visionitem := protocol.AgentPerceptionVisionItem{
				CloseEdge: closeEdge.Clone().SetAngle(closeEdge.Angle() - orientation), // perpendicular to relative position vector, left side
				Center:    centervec,
				FarEdge:   farEdge.Clone().SetAngle(farEdge.Angle() - orientation), // perpendicular to relative position vector, right side
				// FIXME(jerome): /20 here is to convert velocity per second in velocity per tick; should probably handle velocities in m/s everywhere ?
				Velocity: otheragentstate.GetVelocity().Clone().SetAngle(otheragentstate.GetVelocity().Angle() - orientation).Scale(1 / 20),
				Tag:      protocol.AgentPerceptionVisionItemTag.Agent,
			}

			vision = append(vision, visionitem)
		}
	}
	serverstate.Agentsmutex.Unlock()

	return vision
}

func viewObstacles(serverstate *state.ServerState, agentstate entities.AgentState) []protocol.AgentPerceptionVisionItem {

	vision := make([]protocol.AgentPerceptionVisionItem, 0)

	absoluteposition := agentstate.GetPosition()
	orientation := agentstate.GetOrientation()

	radiussq := agentstate.VisionRadius * agentstate.VisionRadius

	// On détermine les bords gauche et droit du cône de vision de l'agent
	halfvisionangle := agentstate.VisionAngle / 2
	leftvisionrelvec := vector.MakeVector2(1, 1).SetMag(agentstate.VisionRadius).SetAngle(orientation + halfvisionangle*-1)
	rightvisionrelvec := vector.MakeVector2(1, 1).SetMag(agentstate.VisionRadius).SetAngle(orientation + halfvisionangle)

	for _, obstacle := range serverstate.MapMemoization.Obstacles {

		edges := make([]vector.Vector2, 0)
		//rejectededges := make([]vector.Vector2, 0)

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

			obstacleperception := protocol.AgentPerceptionVisionItem{
				CloseEdge: closeEdge,
				Center:    relcenteragentaligned,
				FarEdge:   farEdge,
				Velocity:  vector.MakeNullVector2(),
				Tag:       protocol.AgentPerceptionVisionItemTag.Obstacle,
			}

			vision = append(vision, obstacleperception)

		} else if len(edges) > 0 {
			// problems with FOV > 180
		}
	}

	return vision
}
