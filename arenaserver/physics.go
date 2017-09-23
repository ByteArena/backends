package arenaserver

func (server *Server) update() {

	timeStep := 1.0 / float64(server.GetTicksPerSecond())

	server.debugNbUpdates++
	server.debugNbMutations++

	server.ProcessMutations()
	server.game.Step(timeStep)
}
