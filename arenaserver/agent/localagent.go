package agent

import (
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/game/entities"
)

type LocalAgentInterface interface {
	entities.AgentInterface
}

type LocalAgentImp struct {
	entities.AgentImp
	DebugNbPutPerception int
}

func MakeLocalAgentImp() LocalAgentImp {
	return LocalAgentImp{
		AgentImp: entities.MakeAgentImp(),
	}
}

func (agent LocalAgentImp) String() string {
	return "<LocalAgentImp(" + agent.GetId().String() + ")>"
}

func (agent LocalAgentImp) SetPerception(perception protocol.Perception, comm protocol.AgentCommunicatorInterface) error {
	return nil
}
