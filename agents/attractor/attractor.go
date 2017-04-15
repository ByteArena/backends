package attractoragent

import (
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

var speedchange = 0.0
var sign = 1.0

func (agent AttractorAgent) SetPerception(perception state.Perception, comm protocol.AgentCommunicator, agentstate state.AgentState) {

	speed := perception.Specs.MaxSpeed

	if speedchange+0.01 >= speed {
		sign = -1
	} else if speedchange <= 0 {
		sign = 1
	}

	speedchange += sign * 0.01

	desired := utils.MakeVector2(1*sign, 20).SetMag(speed - speedchange)

	steeringx, steeringy := desired.Get()

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
