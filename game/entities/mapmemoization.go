package entities

type MapMemoization struct {
	Obstacles []Obstacle
}

// func initializeMapMemoization(arenaMap *mapcontainer.MapContainer) *MapMemoization {

// 	///////////////////////////////////////////////////////////////////////////
// 	// Obstacles
// 	///////////////////////////////////////////////////////////////////////////

// 	obstacles := make([]entities.Obstacle, 0)

// 	// Obstacles formed by the grounds
// 	for _, ground := range arenaMap.Data.Grounds {
// 		for _, polygon := range ground.Outline {
// 			for i := 0; i < len(polygon.Points)-1; i++ {
// 				a := polygon.Points[i]
// 				b := polygon.Points[i+1]

// 				obstacles = append(obstacles, entities.MakeObstacle(
// 					ground.Id,
// 					entities.ObstacleType.Ground,
// 					vector.MakeVector2(a.X, a.Y),
// 					vector.MakeVector2(b.X, b.Y),
// 				))
// 			}
// 		}
// 	}

// 	// Explicit obstacles
// 	for _, obstacle := range arenaMap.Data.Obstacles {
// 		polygon := obstacle.Polygon
// 		for i := 0; i < len(polygon.Points)-1; i++ {
// 			a := polygon.Points[i]
// 			b := polygon.Points[i+1]
// 			obstacles = append(obstacles, entities.MakeObstacle(
// 				obstacle.Id,
// 				entities.ObstacleType.Object,
// 				vector.MakeVector2(a.X, a.Y),
// 				vector.MakeVector2(b.X, b.Y),
// 			))
// 		}
// 	}

// 	return &MapMemoization{
// 		Obstacles: obstacles,
// 	}
// }
