package agent

import (
	"github.com/netgusto/bytearena/server/protocol"
	"github.com/netgusto/bytearena/server/state"
)

type LocalAgent interface {
	Agent
}

type LocalAgentImp struct {
	AgentImp
	DebugNbPutPerception int
}

func MakeLocalAgentImp() LocalAgentImp {
	return LocalAgentImp{
		AgentImp: MakeAgentImp(),
	}
}

func (agent LocalAgentImp) String() string {
	return "<LocalAgentImp(" + agent.GetId().String() + ")>"
}

func (agent LocalAgentImp) PutPerception(perception state.Perception, server protocol.AgentCommOperator) {
	//perceptionjson, _ := json.Marshal(perception)
	//log.Println("LOCAL AGENT", string(perceptionjson))
}
