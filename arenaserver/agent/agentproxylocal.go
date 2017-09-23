package agent

import (
	"github.com/bytearena/bytearena/arenaserver/protocol"
)

type AgentProxyLocalInterface interface {
	AgentProxyInterface
}

type AgentProxyLocal struct {
	AgentProxyGeneric
	DebugNbPutPerception int
}

func MakeLocalAgentImp() AgentProxyLocal {
	return AgentProxyLocal{
		AgentProxyGeneric: MakeAgentProxyGeneric(),
	}
}

func (agent AgentProxyLocal) String() string {
	return "<LocalAgentImp(" + agent.GetProxyUUID().String() + ")>"
}

func (agent AgentProxyLocal) SetPerception(perception protocol.AgentPerception, comm protocol.AgentCommunicatorInterface) error {
	return nil
}
