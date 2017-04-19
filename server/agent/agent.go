package agent

import (
	"log"
	"math"

	"github.com/netgusto/bytearena/server/protocol"
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/utils"
	uuid "github.com/satori/go.uuid"
)

type Agent interface {
	GetId() uuid.UUID
	String() string
	GetPerception(serverstate *state.ServerState) state.Perception
	SetPerception(perception state.Perception, comm protocol.AgentCommunicator, agentstate state.AgentState) // abstract method
}

type AgentImp struct {
	id uuid.UUID
}

func MakeAgentImp() AgentImp {
	return AgentImp{
		id: uuid.NewV4(), // random uuid
	}
}

func (agent AgentImp) GetId() uuid.UUID {
	return agent.id
}

func (agent AgentImp) String() string {
	return "<AgentImp(" + agent.GetId().String() + ")>"
}

func (agent AgentImp) GetPerception(serverstate *state.ServerState) state.Perception {
	p := state.Perception{}
	agentstate := serverstate.GetAgentState(agent.GetId())

	absoluteposition := agentstate.Position
	orientation := agentstate.Orientation

	p.Internal.Velocity = agentstate.Velocity.Clone()
	p.Internal.Proprioception = agentstate.Radius

	// l'angle d'orientation de l'agent par rapport au "Nord" de l'arène
	p.Internal.Magnetoreception = orientation

	p.Specs.MaxSpeed = agentstate.MaxSpeed
	p.Specs.MaxSteeringForce = agentstate.MaxSteeringForce
	p.Specs.MaxAngularVelocity = agentstate.MaxAngularVelocity

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
				Center:   centervec,
				Radius:   otheragentstate.Radius,
				Velocity: otheragentstate.Velocity.Clone().SetAngle(otheragentstate.Velocity.Angle() - orientation),
				Tag:      otheragentstate.Tag,
			}

			p.External.Vision = append(p.External.Vision, visionitem)
		}
	}
	serverstate.Agentsmutex.Unlock()

	// Vision: les obstacles
	halfvisionangle := agentstate.VisionAngle / 2
	leftvisionrelvec := utils.MakeVector2(1, 1).SetMag(agentstate.VisionRadius).SetAngle(orientation + halfvisionangle*-1)
	rightvisionrelvec := utils.MakeVector2(1, 1).SetMag(agentstate.VisionRadius).SetAngle(orientation + halfvisionangle)

	serverstate.Obstaclesmutex.Lock()
	for _, obstacle := range serverstate.Obstacles {

		edges := make([]utils.Vector2, 0)

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
			if relAngleA > math.Pi { // 180° en radians
				relAngleA -= math.Pi * 2 // 360° en radian
			}

			if math.Abs(relAngleA) <= halfvisionangle {
				// point dans le champ de vision !
				edges = append(edges, relvecA.Add(absoluteposition))
			}
		}

		if distsqB <= radiussq {
			absAngleB := relvecB.Angle()
			relAngleB := absAngleB - orientation

			// On passe de 0° / 360° à -180° / +180°
			if relAngleB > math.Pi { // 180° en radians
				relAngleB -= math.Pi * 2 // 360° en radian
			}

			if math.Abs(relAngleB) <= halfvisionangle {
				// point dans le champ de vision !
				edges = append(edges, relvecB.Add(absoluteposition))
			}
		}

		{
			// Sur les bords de la perception
			// http://www.wyrmtale.com/blog/2013/115/2d-line-intersection-in-c
			if point, intersects, colinear, _ := leftvisionrelvec.IntersectionWithLineSegment(relvecA, relvecB); intersects && !colinear {
				//log.Println("INTERSECT LEFT", point)
				//serverstate.DebugIntersects = append(serverstate.DebugIntersects, point.Add(absoluteposition))
				edges = append(edges, point.Add(absoluteposition))
			}

			if point, intersects, colinear, _ := rightvisionrelvec.IntersectionWithLineSegment(relvecA, relvecB); intersects && !colinear {
				//log.Println("INTERSECT RIGHT", point)
				//serverstate.DebugIntersects = append(serverstate.DebugIntersects, point.Add(absoluteposition))
				edges = append(edges, point.Add(absoluteposition))
			}
		}

		{
			// Sur l'horizon de perception (arc de cercle)
			intersections := utils.CircleLineCollision(
				relvecA,
				relvecB,
				utils.MakeVector2(0, 0),
				agentstate.VisionRadius,
			)

			for _, point := range intersections {
				// il faut vérifier que le point se trouve bien sur le segment
				// il faut vérifier que l'angle du point de collision se trouve bien dans le champ de vision de l'agent

				if point.PointOnLineSegment(relvecA, relvecB) {
					relvecangle := point.Angle() - orientation

					// On passe de 0° / 360° à -180° / +180°
					if relvecangle > math.Pi { // 180° en radians
						relvecangle -= math.Pi * 2 // 360° en radian
					}

					if math.Abs(relvecangle) <= halfvisionangle {
						edges = append(edges, point.Add(absoluteposition))
						//serverstate.DebugIntersects = append(serverstate.DebugIntersects, point.Add(absoluteposition))
					}
				}
			}
		}

		if len(edges) == 2 {
			edgeone := edges[0]
			edgetwo := edges[1]
			center := edgetwo.Add(edgeone).DivScalar(2)

			visiblemag := edgetwo.Sub(edgeone).Mag()

			relcenter := center.Sub(absoluteposition) // aligned on north
			relcenterangle := relcenter.Angle()
			relcenteragentaligned := relcenter.SetAngle(relcenterangle - orientation)

			obstacleperception := state.PerceptionVisionItem{
				Center:   relcenteragentaligned,
				Radius:   visiblemag / 2,
				Velocity: utils.MakeVector2(0, 0),
				Tag:      "obstacle",
			}

			p.External.Vision = append(p.External.Vision, obstacleperception)

		} else if len(edges) > 0 {
			log.Println("NOPE !", edges)
		}

		for _, edge := range edges {
			serverstate.DebugIntersects = append(serverstate.DebugIntersects, edge)
		}
	}
	serverstate.Obstaclesmutex.Unlock()

	return p
}

func (agent AgentImp) SetPerception(perception state.Perception, comm protocol.AgentCommunicator, agentstate state.AgentState) {
	// I'm abstract, override me !
}
