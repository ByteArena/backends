package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/bytearena/bytearena/arenaserver"
	"github.com/bytearena/bytearena/arenaserver/state"
	"github.com/bytearena/bytearena/common/mq"
	"github.com/bytearena/bytearena/common/types"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/leakybucket"
)

func StreamState(srv *arenaserver.Server, brokerclient mq.ClientInterface, arenaServerUUID string) {

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
					srv.GetGame(),
					serverstate,
					arenaServerUUID,
				)

				json, err := json.Marshal(msg)
				if err != nil {
					utils.Debug("viz-server", "json error, wtf")
					fmt.Println("")
					fmt.Println("")
					fmt.Println("")
					fmt.Println("")
					fmt.Println("")
					fmt.Println("")
					fmt.Println("")
					fmt.Println(err)
					fmt.Println(msg)
					fmt.Println("")
					fmt.Println("")
					fmt.Println("")
					fmt.Println("")
					fmt.Println("")
					fmt.Println("")
					fmt.Println("")
					return
				}

				buk.AddFrame(string(json))
			}
		}
	}

}

func transformServerStateToVizMessage(game arenaserver.GameInterface, state state.ServerState, arenaServerUUID string) types.VizMessage {

	msg := types.VizMessage{
		GameID:          game.GetId(),
		ArenaServerUUID: arenaServerUUID,
	}

	state.Projectilesmutex.Lock()
	for _, projectile := range state.Projectiles {
		msg.Projectiles = append(msg.Projectiles, types.VizProjectileMessage{
			Id:       projectile.Id,
			Position: projectile.GetPosition(),
			Velocity: projectile.GetVelocity(),
			Kind:     "projectile",
		})
	}
	state.Projectilesmutex.Unlock()

	state.Agentsmutex.Lock()
	for id, agent := range state.Agents {
		msg.Agents = append(msg.Agents, types.VizAgentMessage{
			Id:           id,
			Name:         agent.GetName(),
			Kind:         "agent",
			Position:     agent.GetPosition(),
			Velocity:     agent.GetVelocity(),
			Radius:       agent.GetRadius(),
			Orientation:  agent.GetOrientation(),
			VisionRadius: agent.VisionRadius,
			VisionAngle:  agent.VisionAngle,
			DebugMsg:     agent.DebugMsg,
		})
	}
	state.Agentsmutex.Unlock()

	msg.DebugPoints = state.DebugPoints

	return msg
}
