package agent

import (
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/arenaserver/state"
)

type LocalAgentInterface interface {
	AgentInterface
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

func (agent LocalAgentImp) SetPerception(perception state.Perception, comm protocol.AgentCommunicatorInterface) error {
	return nil
}
