package protocol

import (
	"encoding/json"
	"log"

	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/leakybucket"
)

func StreamState(srv *arenaserver.Server, brokerclient mq.ClientInterface) {

	buk := leakybucket.NewBucket(srv.GetTicksPerSecond(), 10, func(batch leakybucket.Batch, bucket *leakybucket.Bucket) {
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

func transformServerStateToVizMessage(arenaid string, state state.ServerState) types.VizMessage {

	msg := types.VizMessage{
		ArenaId: arenaid,
	}

	state.Projectilesmutex.Lock()
	for _, projectile := range state.Projectiles {
		msg.Projectiles = append(msg.Projectiles, types.VizProjectileMessage{
			Position: projectile.Velocity,
			Radius:   projectile.Radius,
			Kind:     "projectiles",
			From: types.VizAgentMessage{
				Position: projectile.Position,
			},
		})
	}
	state.Projectilesmutex.Unlock()

	state.Agentsmutex.Lock()
	for id, agent := range state.Agents {
		msg.Agents = append(msg.Agents, types.VizAgentMessage{
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
		msg.Obstacles = append(msg.Obstacles, types.VizObstacleMessage{
			Id: obstacle.Id,
			A:  obstacle.GetA(),
			B:  obstacle.GetB(),
		})
	}
	state.Obstaclesmutex.Unlock()

	msg.DebugIntersects = state.DebugIntersects
	msg.DebugIntersectsRejected = state.DebugIntersectsRejected
	msg.DebugPoints = state.DebugPoints

	return msg
}
