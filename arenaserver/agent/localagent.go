package agent

import (
	"github.com/bytearena/bytearena/arenaserver/protocol"
	agentstate "github.com/bytearena/bytearena/arenaserver/state/agent"
	stateprotocol "github.com/bytearena/bytearena/arenaserver/state/protocol"
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

func (agent LocalAgentImp) SetPerception(perception agentstate.Perception, comm protocol.AgentCommunicator, agentstate stateprotocol.AgentStateInterface) {
}
