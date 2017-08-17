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

func StreamState(srv *arenaserver.Server, brokerclient mq.ClientInterface, UUID string) {

	buk := leakybucket.NewBucket(
		srv.GetTicksPerSecond(),
		5, // keep 5 seconds of stream in buffer
		func(batch leakybucket.Batch, bucket *leakybucket.Bucket) {
			frames := batch.GetFrames()
			jsonbatch := make([]json.RawMessage, len(frames))
			for i, frame := range frames {
				jsonbatch[i] = json.RawMessage(frame.GetPayload())
			}

			brokerclient.Publish("viz", "message", jsonbatch)
		},
	)

	stateobserver := srv.SubscribeStateObservation()
	for {
		select {
		case serverstate := <-stateobserver:
			{
				msg := transformServerStateToVizMessage(
					srv.GetArena(),
					serverstate,
					UUID,
				)

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

func transformServerStateToVizMessage(game arenaserver.Game, state state.ServerState, UUID string) types.VizMessage {

	msg := types.VizMessage{
		ArenaId: game.GetId(),
		UUID:    UUID,
	}

	state.Projectilesmutex.Lock()
	for _, projectile := range state.Projectiles {
		msg.Projectiles = append(msg.Projectiles, types.VizProjectileMessage{
			Id:       projectile.Id,
			Position: projectile.Position,
			Velocity: projectile.Velocity,
			Kind:     "projectile",
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

	msg.DebugPoints = state.DebugPoints

	return msg
}
