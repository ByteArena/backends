package protocol

import (
	"encoding/json"
	"log"

	"github.com/bytearena/bytearena/common/messagebroker"
	"github.com/bytearena/bytearena/leakybucket"
	"github.com/bytearena/bytearena/server"
	"github.com/bytearena/bytearena/server/state"
)

func StreamState(srv *server.Server, brokerclient *messagebroker.Client) {

	buk := leakybucket.NewBucket(srv.GetTicksPerSecond(), 10, func(batch leakybucket.Batch, bucket *leakybucket.Bucket) {

		log.Println("batch !")
		frames := batch.GetFrames()
		jsonbatch := make([]json.RawMessage, len(frames))
		for i, frame := range frames {
			jsonbatch[i] = json.RawMessage(frame.GetPayload())
		}

		brokerclient.Publish("viz", "message", jsonbatch)
	})

	stateobserver := srv.SubscribeStateObservation()
	for {
		select {
		case state := <-stateobserver:
			{
				msg := transformServerStateToVizMessage(srv.GetArena().GetId(), state)

				json, err := json.Marshal(msg)
				if err != nil {
					log.Println("json error, wtf")
					return
				}

				buk.AddFrame(string(json))
			}
		}
	}

}

func transformServerStateToVizMessage(arenaid string, state state.ServerState) VizMessage {

	msg := VizMessage{
		ArenaId: arenaid,
	}

	state.Projectilesmutex.Lock()
	for _, projectile := range state.Projectiles {
		msg.Projectiles = append(msg.Projectiles, VizProjectileMessage{
			Position: projectile.Velocity,
			Radius:   projectile.Radius,
			Kind:     "projectiles",
			From: VizAgentMessage{
				Position: projectile.Position,
			},
		})
	}
	state.Projectilesmutex.Unlock()

	state.Agentsmutex.Lock()
	for id, agent := range state.Agents {
		msg.Agents = append(msg.Agents, VizAgentMessage{
			Id:           id,
			Kind:         "agent",
			Position:     agent.Position,
			Velocity:     agent.Velocity,
			Radius:       agent.Radius,
			Orientation:  agent.Orientation,
			VisionRadius: agent.VisionRadius,
			VisionAngle:  agent.VisionAngle,
		})
	}
	state.Agentsmutex.Unlock()

	state.Obstaclesmutex.Lock()
	for _, obstacle := range state.Obstacles {
		msg.Obstacles = append(msg.Obstacles, VizObstacleMessage{
			Id: obstacle.Id,
			A:  obstacle.A,
			B:  obstacle.B,
		})
	}
	state.Obstaclesmutex.Unlock()

	msg.DebugIntersects = state.DebugIntersects
	msg.DebugIntersectsRejected = state.DebugIntersectsRejected
	msg.DebugPoints = state.DebugPoints

	return msg
}
