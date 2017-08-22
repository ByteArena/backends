package container

import uuid "github.com/satori/go.uuid"

type AgentContainer struct {
	AgentId     uuid.UUID
	containerid ContainerId
	ImageName   string
}

func MakeAgentContainer(agentid uuid.UUID, containerid ContainerId, imageName string) AgentContainer {
	return AgentContainer{
		AgentId:     agentid,
		containerid: containerid,
		ImageName:   imageName,
	}
}
