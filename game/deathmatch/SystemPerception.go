package deathmatch

import (
	"math"
	"sync"

	commontypes "github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils/trigo"
	"github.com/bytearena/bytearena/common/utils/vector"

	"github.com/bytearena/box2d"

	"github.com/bytearena/bytearena/common/types/mapcontainer"
	"github.com/bytearena/ecs"
)

// https://legends2k.github.io/2d-fov/design.html
// http://ncase.me/sight-and-light/

func systemPerception(deathmatch *DeathmatchGame) {
	entitiesWithPerception := deathmatch.perceptorsView.Get()
	wg := sync.WaitGroup{}
	wg.Add(len(entitiesWithPerception))

	for _, entityResult := range entitiesWithPerception {
		perceptionAspect := entityResult.Components[deathmatch.perceptionComponent].(*Perception)
		go func(perceptionAspect *Perception, entity *ecs.Entity, wg *sync.WaitGroup) {
			perceptionAspect.SetPerception(computeAgentPerception(
				deathmatch,
				deathmatch.gameDescription.GetMapContainer(),
				entity.GetID(),
			))
			wg.Done()
		}(perceptionAspect, entityResult.Entity, &wg)
	}

	wg.Wait()
}

func computeAgentPerception(game *DeathmatchGame, arenaMap *mapcontainer.MapContainer, entityid ecs.EntityID) []byte {
	//watch := utils.MakeStopwatch("computeAgentPerception()")
	//watch.Start("global")

	p := agentPerception{}

	entityresult := game.getEntity(entityid,
		game.physicalBodyComponent,
		game.steeringComponent,
		game.perceptionComponent,
	)

	if entityresult == nil {
		return []byte{}
	}

	physicalAspect := entityresult.Components[game.physicalBodyComponent].(*PhysicalBody)
	steeringAspect := entityresult.Components[game.steeringComponent].(*Steering)
	perceptionAspect := entityresult.Components[game.perceptionComponent].(*Perception)

	orientation := physicalAspect.GetOrientation()
	velocity := physicalAspect.GetVelocity()
	radius := physicalAspect.GetRadius()

	// FIXME(jerome): 1/20
	p.Internal.Velocity = velocity.Clone().SetAngle(velocity.Angle() - orientation).Scale(1.0 / 20.0)
	p.Internal.Proprioception = radius
	p.Internal.Magnetoreception = orientation // l'angle d'orientation de l'agent par rapport au "Nord" de l'arène

	p.Specs.MaxSpeed = physicalAspect.GetMaxSpeed()
	p.Specs.MaxSteeringForce = steeringAspect.GetMaxSteeringForce()
	p.Specs.MaxAngularVelocity = physicalAspect.GetMaxAngularVelocity()
	p.Specs.DragForce = physicalAspect.GetDragForce()
	p.Specs.VisionRadius = perceptionAspect.GetVisionRadius()
	p.Specs.VisionAngle = perceptionAspect.GetVisionAngle()

	//watch.Start("p.External.Vision =")
	p.External.Vision = computeAgentVision(game, entityresult.Entity, physicalAspect, perceptionAspect)
	//watch.Stop("p.External.Vision =")

	//watch.Start("json.Marshal")
	res, _ := p.MarshalJSON()
	//log.Println("JSON SIZE ", float64(len(res))/1024.0, "KB")
	//res := []byte("{\"Internal\":{\"Velocity\":[0,0]},\"Specs\":{\"VisionRadius\":1},\"External\":{\"Vision\":[]}}")
	//watch.Stop("json.Marshal")
	//watch.Stop("global")
	//fmt.Println(watch.String())

	return res
}

func computeAgentVision(game *DeathmatchGame, entity *ecs.Entity, physicalAspect *PhysicalBody, perceptionAspect *Perception) []agentPerceptionVisionItem {

	vision := make([]agentPerceptionVisionItem, 0)

	vision = append(vision, viewEntities(game, entity, physicalAspect, perceptionAspect)...)

	// on met la vision à l'échelle de l'agent
	for i, visionItem := range vision {
		visionItem.Center = visionItem.Center.Transform(game.physicalToAgentSpaceTransform)
		visionItem.FarEdge = visionItem.FarEdge.Transform(game.physicalToAgentSpaceTransform)
		visionItem.NearEdge = visionItem.NearEdge.Transform(game.physicalToAgentSpaceTransform)
		visionItem.Velocity = visionItem.Velocity.Transform(game.physicalToAgentSpaceTransform)
		vision[i] = visionItem
	}

	return vision
}

func viewEntities(game *DeathmatchGame, entity *ecs.Entity, physicalAspect *PhysicalBody, perceptionAspect *Perception) []agentPerceptionVisionItem {

	//watch := utils.MakeStopwatch("viewEntities()")
	//watch.Start("global")

	vision := make([]agentPerceptionVisionItem, 0)

	// for _, entityresult := range game.physicalView.Get() {
	// 	physicalAspect := entityresult.Components[game.physicalBodyComponent].(*PhysicalBody)
	// 	if physicalAspect.GetVelocity().Mag() > 0.01 {
	// 		physicalAspect.SetOrientation(physicalAspect.GetVelocity().Angle())
	// 	}
	// }

	pi2 := math.Pi * 2
	halfpi := math.Pi / 2
	threepi2 := math.Pi + halfpi

	agentPosition := physicalAspect.GetPosition()
	agentOrientation := physicalAspect.GetOrientation()
	visionAngle := perceptionAspect.GetVisionAngle()
	visionRadius := perceptionAspect.GetVisionRadius()
	visionRadiusSq := visionRadius * visionRadius

	halfVisionAngle := visionAngle / 2
	leftVisionEdgeAngle := math.Mod(agentOrientation-halfVisionAngle, pi2)
	rightVisionEdgeAngle := math.Mod(agentOrientation+halfVisionAngle, pi2)
	leftVisionRelvec := vector.MakeVector2(1, 1).SetMag(visionRadius).SetAngle(leftVisionEdgeAngle)
	rightVisionRelvec := vector.MakeVector2(1, 1).SetMag(visionRadius).SetAngle(rightVisionEdgeAngle)

	// Determine View cone AABB

	notableVisionConePoints := make([]vector.Vector2, 0)
	notableVisionConePoints = append(notableVisionConePoints, agentPosition)                        // center
	notableVisionConePoints = append(notableVisionConePoints, leftVisionRelvec.Add(agentPosition))  // left radius
	notableVisionConePoints = append(notableVisionConePoints, rightVisionRelvec.Add(agentPosition)) // right radius

	minAngle := math.Min(leftVisionEdgeAngle, rightVisionEdgeAngle)
	maxAngle := math.Max(leftVisionEdgeAngle, rightVisionEdgeAngle)

	if minAngle <= 0 && maxAngle > 0 {
		// Determine north point on circle
		notableVisionConePoints = append(notableVisionConePoints,
			vector.MakeVector2(1, 1).SetMag(visionRadius).SetAngle(0).Add(agentPosition),
		)
	}

	if minAngle <= halfpi && maxAngle > halfpi {
		// Determine east point on circle
		notableVisionConePoints = append(notableVisionConePoints,
			vector.MakeVector2(1, 1).SetMag(visionRadius).SetAngle(halfpi).Add(agentPosition),
		)
	}

	if minAngle <= math.Pi && maxAngle > math.Pi {
		// Determine south point on circle
		notableVisionConePoints = append(notableVisionConePoints,
			vector.MakeVector2(1, 1).SetMag(visionRadius).SetAngle(math.Pi).Add(agentPosition),
		)
	}

	if minAngle <= (threepi2) && maxAngle > (threepi2) {
		// Determine west point on circle
		notableVisionConePoints = append(notableVisionConePoints,
			vector.MakeVector2(1, 1).SetMag(visionRadius).SetAngle(threepi2).Add(agentPosition),
		)
	}

	entityAABB := vector.GetAABBForPointList(notableVisionConePoints...)
	elementsInAABB := make(map[ecs.EntityID]commontypes.PhysicalBodyDescriptor)

	game.PhysicalWorld.QueryAABB(func(fixture *box2d.B2Fixture) bool {
		if descriptor, ok := fixture.GetBody().GetUserData().(commontypes.PhysicalBodyDescriptor); ok {
			//elementsInAABB = append(elementsInAABB, descriptor)
			if _, isInMap := elementsInAABB[descriptor.ID]; !isInMap {
				elementsInAABB[descriptor.ID] = descriptor
			}
		}
		return true // keep going to find all fixtures in the query area
	}, entityAABB.ToB2AABB())

	//log.Println("AABB:", len(elementsInAABB))

	for _, bodyDescriptor := range elementsInAABB {

		if bodyDescriptor.ID == entity.ID {
			// one does not see itself
			continue
		}

		if bodyDescriptor.Type == commontypes.PhysicalBodyDescriptorType.Agent || bodyDescriptor.Type == commontypes.PhysicalBodyDescriptorType.Projectile {

			visionType := agentPerceptionVisionItemTag.Obstacle
			switch bodyDescriptor.Type {
			case commontypes.PhysicalBodyDescriptorType.Agent:
				visionType = agentPerceptionVisionItemTag.Agent
			case commontypes.PhysicalBodyDescriptorType.Obstacle:
				visionType = agentPerceptionVisionItemTag.Obstacle
			case commontypes.PhysicalBodyDescriptorType.Projectile:
				visionType = agentPerceptionVisionItemTag.Projectile
			case commontypes.PhysicalBodyDescriptorType.Ground:
				visionType = agentPerceptionVisionItemTag.Obstacle
			default:
				continue
			}

			//log.Println("Circle", bodyDescriptor.Type)
			// view a circle

			if bodyDescriptor.Type == commontypes.PhysicalBodyDescriptorType.Projectile {
				ownedQr := game.getEntity(bodyDescriptor.ID, game.ownedComponent)
				if ownedQr != nil {
					ownedAspect := ownedQr.Components[game.ownedComponent].(*Owned)
					if ownedAspect.GetOwner() == entity.GetID() {
						// do not show projectiles to their sender
						continue
					}
				}
			}

			otherQr := game.getEntity(bodyDescriptor.ID, game.physicalBodyComponent)
			otherPhysicalAspect := otherQr.Components[game.physicalBodyComponent].(*PhysicalBody)

			otherPosition := otherPhysicalAspect.GetPosition()
			otherVelocity := otherPhysicalAspect.GetVelocity()
			otherRadius := otherPhysicalAspect.GetRadius()

			if otherPosition.Equals(agentPosition) {
				// bodies have the exact same position; should never happen
				continue
			}

			centervec := otherPosition.Sub(agentPosition)
			centersegment := vector.MakeSegment2(vector.MakeNullVector2(), centervec)
			agentdiameter := centersegment.OrthogonalToBCentered().SetLengthFromCenter(otherRadius * 2)

			nearEdge, farEdge := agentdiameter.Get()

			distsq := centervec.MagSq()
			if distsq <= visionRadiusSq {

				// Il faut aligner l'angle du vecteur sur le heading courant de l'agent
				centervec = centervec.SetAngle(centervec.Angle() - agentOrientation)

				visionitem := agentPerceptionVisionItem{
					NearEdge: nearEdge.Clone().SetAngle(nearEdge.Angle() - agentOrientation), // perpendicular to relative position vector, left side
					Center:   centervec,
					FarEdge:  farEdge.Clone().SetAngle(farEdge.Angle() - agentOrientation), // perpendicular to relative position vector, right side
					// FIXME(jerome): /20 here is to convert velocity per second in velocity per tick; should probably handle velocities in m/s everywhere ?
					Velocity: otherVelocity.Clone().Scale(1.0 / 20.0).SetAngle(otherVelocity.Angle() - agentOrientation),
					Tag:      visionType,
				}

				vision = append(vision, visionitem)
			}
		} else {

			// view a polygon
			//rejectededges := make([]vector.Vector2, 0)

			otherQr := game.getEntity(bodyDescriptor.ID, game.physicalBodyComponent)
			otherPhysicalAspect := otherQr.Components[game.physicalBodyComponent].(*PhysicalBody)

			//bodyPoly := otherPhysicalAspect.body.GetFixtureList().GetShape().(*box2d.B2ChainShape)
			//vertices := bodyPoly.M_vertices

			vertices := make([]box2d.B2Vec2, 0)
			fixture := otherPhysicalAspect.body.GetFixtureList()
			for fixture != nil {
				edge := fixture.GetShape().(*box2d.B2EdgeShape)
				vertices = append(vertices, edge.M_vertex1, edge.M_vertex2)
				fixture = fixture.M_next
			}

			for i := 1; i < len(vertices); i++ {

				edges := make([]vector.Vector2, 0)

				pointA := vector.FromB2Vec2(vertices[i-1])
				pointB := vector.FromB2Vec2(vertices[i])

				segmentAABB := vector.GetAABBForPointList(pointA, pointB)
				if !segmentAABB.Overlaps(entityAABB) {
					continue
				}

				relvecA := pointA.Sub(agentPosition)
				relvecB := pointB.Sub(agentPosition)

				distsqA := relvecA.MagSq()
				distsqB := relvecB.MagSq()

				// Comment déterminer si le vecteur entre dans le champ de vision ?
				// => Intersection entre vecteur et segment gauche, droite

				if distsqA <= visionRadiusSq {
					// in radius
					absAngleA := relvecA.Angle()
					relAngleA := absAngleA - agentOrientation

					// On passe de 0° / 360° à -180° / +180°
					relAngleA = trigo.FullCircleAngleToSignedHalfCircleAngle(relAngleA)

					if math.Abs(relAngleA) <= halfVisionAngle {
						// point dans le champ de vision !
						edges = append(edges, relvecA.Add(agentPosition))
					} else {
						//rejectededges = append(rejectededges, relvecA.Add(absoluteposition))
					}
				}

				if distsqB <= visionRadiusSq {
					absAngleB := relvecB.Angle()
					relAngleB := absAngleB - agentOrientation

					// On passe de 0° / 360° à -180° / +180°
					relAngleB = trigo.FullCircleAngleToSignedHalfCircleAngle(relAngleB)

					if math.Abs(relAngleB) <= halfVisionAngle {
						// point dans le champ de vision !
						edges = append(edges, relvecB.Add(agentPosition))
					} else {
						//rejectededges = append(rejectededges, relvecB.Add(absoluteposition))
					}
				}

				{
					// Sur les bords de la perception
					if point, intersects, colinear, _ := trigo.IntersectionWithLineSegment(vector.MakeNullVector2(), leftVisionRelvec, relvecA, relvecB); intersects && !colinear {
						// INTERSECT LEFT
						edges = append(edges, point.Add(agentPosition))
					}

					if point, intersects, colinear, _ := trigo.IntersectionWithLineSegment(vector.MakeNullVector2(), rightVisionRelvec, relvecA, relvecB); intersects && !colinear {
						// INTERSECT RIGHT
						edges = append(edges, point.Add(agentPosition))
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
							relvecangle := point.Angle() - agentOrientation

							// On passe de 0° / 360° à -180° / +180°
							relvecangle = trigo.FullCircleAngleToSignedHalfCircleAngle(relvecangle)

							if math.Abs(relvecangle) <= halfVisionAngle {
								edges = append(edges, point.Add(agentPosition))
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

					relCenter := center.Sub(agentPosition) // aligned on north
					relCenterAngle := relCenter.Angle()
					relCenterAgentAligned := relCenter.SetAngle(relCenterAngle - agentOrientation)

					relEdgeOne := edgeone.Sub(agentPosition)
					relEdgeTwo := edgetwo.Sub(agentPosition)

					relEdgeOneAgentAligned := relEdgeOne.SetAngle(relEdgeOne.Angle() - agentOrientation)
					relEdgeTwoAgentAligned := relEdgeTwo.SetAngle(relEdgeTwo.Angle() - agentOrientation)

					var nearEdge, farEdge vector.Vector2
					if relEdgeTwoAgentAligned.MagSq() > relEdgeOneAgentAligned.MagSq() {
						nearEdge = relEdgeOneAgentAligned
						farEdge = relEdgeTwoAgentAligned
					} else {
						nearEdge = relEdgeTwoAgentAligned
						farEdge = relEdgeOneAgentAligned
					}

					obstacleperception := agentPerceptionVisionItem{
						NearEdge: nearEdge,
						Center:   relCenterAgentAligned,
						FarEdge:  farEdge,
						Velocity: vector.MakeNullVector2(),
						Tag:      agentPerceptionVisionItemTag.Obstacle,
					}

					vision = append(vision, obstacleperception)

				} else if len(edges) > 0 {
					// problems with FOV > 180
					//log.Println("SOMETHING'S WRONG !!!!!!!!!!!!!!!!!!!", len(edges))
				}
			}
		}
	}

	vision = processOcclusions(vision, agentPosition)

	renderQr := game.getEntity(entity.ID, game.renderComponent)
	if renderQr != nil {
		renderAspect := renderQr.Components[game.renderComponent].(*Render)
		renderAspect.DebugPoints = make([][2]float64, len(vision))
		for _, v := range vision {

			//absCenter := v.Center.SetAngle(v.Center.Angle() + agentOrientation).Add(agentPosition)
			absNearEdge := v.NearEdge.SetAngle(v.NearEdge.Angle() + agentOrientation).Add(agentPosition)
			absFarEdge := v.FarEdge.SetAngle(v.FarEdge.Angle() + agentOrientation).Add(agentPosition)

			renderAspect.DebugPoints = append(renderAspect.DebugPoints,
				absNearEdge.ToFloatArray(),
				//absCenter.ToFloatArray(),
				absFarEdge.ToFloatArray(),
			)
		}

		renderAspect.DebugPoints = append(renderAspect.DebugPoints,
			//agentPosition.ToFloatArray(),
			leftVisionRelvec.Add(agentPosition).ToFloatArray(),
			rightVisionRelvec.Add(agentPosition).ToFloatArray(),
		)
	}

	//watch.Stop("global")
	//fmt.Println(watch.String())

	return vision
}

type occlusionItem struct {
	visionItem     agentPerceptionVisionItem
	angleRealFrom  float64
	angleRealTo    float64
	angleRatioFrom float64
	angleRatioTo   float64
	distanceSq     float64
}

type byAngleRatio []occlusionItem

func (a byAngleRatio) Len() int           { return len(a) }
func (a byAngleRatio) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byAngleRatio) Less(i, j int) bool { return a[i].angleRatioFrom < a[j].angleRatioFrom }

func processOcclusions(vision []agentPerceptionVisionItem, agentPosition vector.Vector2) []agentPerceptionVisionItem {
	return vision

	// segments := make([]*visibility2d.Segment, len(vision))

	// for i, visionItem := range vision {
	// 	segments[i] = visibility2d.NewSegment(
	// 		visionItem.NearEdge.GetX(),
	// 		visionItem.NearEdge.GetY(),
	// 		visionItem.FarEdge.GetX(),
	// 		visionItem.FarEdge.GetY(),
	// 		visionItem,
	// 	)
	// }

	// pov := visibility2d.Point{agentPosition.GetX(), agentPosition.GetY()}

	// begin := time.Now()
	// visibility := visibility2d.CalculateVisibility(pov, segments)
	// fmt.Println("Took ", float64(time.Now().UnixNano()-begin.UnixNano())/1000000.0, "ms")

	// realVision := make([]agentPerceptionVisionItem, len(visibility))

	// for i, v := range visibility {
	// 	agentVisionItem := v.Userdata.(agentPerceptionVisionItem)
	// 	agentVisionItem.NearEdge = v.Visible.GetPointA()
	// 	agentVisionItem.FarEdge = v.Visible.GetPointB()
	// 	agentVisionItem.Center = v.Visible.Center()
	// 	realVision[i] = agentVisionItem
	// }

	// return realVision
}

func getCircleSegmentAABB(center vector.Vector2, radius float64, angleARad float64, angleBRad float64) (lowerBound vector.Vector2, upperBound vector.Vector2) {
	return vector.MakeVector2(0, 0), vector.MakeVector2(0, 0)
}
