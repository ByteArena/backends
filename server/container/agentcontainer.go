package container

import uuid "github.com/satori/go.uuid"

type AgentContainer struct {
	AgentId     uuid.UUID
	containerid ContainerId
}

func MakeAgentContainer(agentid uuid.UUID, containerid ContainerId) AgentContainer {
	return AgentContainer{
		AgentId:     agentid,
		containerid: containerid,
	}
}
