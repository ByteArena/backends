package entities

import (
	"github.com/bytearena/bytearena/arenaserver/protocol"
	uuid "github.com/satori/go.uuid"
)

type AgentInterface interface {
	GetId() uuid.UUID
	String() string
	SetPerception(perception protocol.AgentPerception, comm protocol.AgentCommunicatorInterface) error // abstract method
}

type AgentImp struct {
	id uuid.UUID
}

func MakeAgentImp() AgentImp {
	return AgentImp{
		id: uuid.NewV4(), // random uuid
	}
}

func (agent AgentImp) GetId() uuid.UUID {
	return agent.id
}

func (agent AgentImp) String() string {
	return "<AgentImp(" + agent.GetId().String() + ")>"
}

func (agent AgentImp) SetPerception(perception protocol.AgentPerception, comm protocol.AgentCommunicatorInterface) error {
	// I'm abstract, override me !
	return nil
}
