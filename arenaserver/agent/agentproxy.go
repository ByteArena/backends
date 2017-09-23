package agent

import (
	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/ecs"
	uuid "github.com/satori/go.uuid"
)

type AgentProxyInterface interface {
	GetProxyUUID() uuid.UUID
	GetEntityId() ecs.EntityID
	SetPerception(perception protocol.AgentPerception, comm protocol.AgentCommunicatorInterface) error // abstract method
	String() string
}

type AgentProxyGeneric struct {
	proxyUUID uuid.UUID
	entityID  ecs.EntityID
}

func MakeAgentProxyGeneric() AgentProxyGeneric {
	return AgentProxyGeneric{
		proxyUUID: uuid.NewV4(), // random uuid
	}
}

func (agent AgentProxyGeneric) GetProxyUUID() uuid.UUID {
	return agent.proxyUUID
}

func (agent *AgentProxyGeneric) SetEntityId(id ecs.EntityID) {
	agent.entityID = id
}

func (agent AgentProxyGeneric) GetEntityId() ecs.EntityID {
	return agent.entityID
}

func (agent AgentProxyGeneric) String() string {
	return "<AgentImp(" + agent.GetProxyUUID().String() + ")>"
}

func (agent AgentProxyGeneric) SetPerception(perception protocol.AgentPerception, comm protocol.AgentCommunicatorInterface) error {
	// I'm abstract, override me !
	return nil
}
