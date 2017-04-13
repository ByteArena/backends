package attractoragent

import (
	"math"

	"github.com/netgusto/bytearena/server/agent"
	"github.com/netgusto/bytearena/server/protocol"
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/utils"
)

type AttractorAgent struct {
	agent.LocalAgentImp
	pincenter utils.Vector2
}

func MakeAttractorAgent() AttractorAgent {
	pin := utils.MakeVector2(400, 300)
	return AttractorAgent{
		LocalAgentImp: agent.MakeLocalAgentImp(),
		pincenter:     pin,
	}
}

var count int

func (agent AttractorAgent) SetPerception(perception state.Perception, comm protocol.AgentCommunicator, agentstate state.AgentState) {

	//log.Println(perception)

	curvelocity := perception.Internal.Velocity
	speed := perception.Specs.MaxSpeed

	// update attractor
	centerx, centery := agent.pincenter.Get()
	radius := 130.0

	absdesiredx := centerx + radius*math.Cos(float64(count)/54.0)
	absdesiredy := centery + radius*math.Sin(float64(count)/54.0)

	count++

	desired := utils.MakeVector2(absdesiredx, absdesiredy).Sub(agentstate.Position)

	disttotarget := desired.Mag()

	if disttotarget < perception.Internal.Proprioception {
		// arrival, slow down
		speed = utils.Map(disttotarget, 0, perception.Internal.Proprioception, 0, perception.Specs.MaxSpeed)
	}

	desired = desired.SetMag(speed)
	steering := desired.Sub(curvelocity).Limit(perception.Specs.MaxSteeringForce)

	steeringx, steeringy := steering.Get()

	mutations := make([]protocol.MessageMutationImp, 1)
	mutations[0] = protocol.MessageMutationImp{
		Method:    "steer",
		Arguments: []byte("[" + utils.FloatToStr(steeringx, 5) + ", " + utils.FloatToStr(steeringy, 5) + "]"),
	}

	comm.PushMutationBatch(protocol.StateMutationBatch{
		AgentId:   agent.GetId(),
		Mutations: mutations,
	})
}
