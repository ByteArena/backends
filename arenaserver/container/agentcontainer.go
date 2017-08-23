package container

import uuid "github.com/satori/go.uuid"

type AgentContainer struct {
	AgentId     uuid.UUID
	containerid ContainerId
	ImageName   string
	IPAddress   string
}

func NewAgentContainer(agentid uuid.UUID, containerid ContainerId, imageName string) *AgentContainer {
	return &AgentContainer{
		AgentId:     agentid,
		containerid: containerid,
		ImageName:   imageName,
		IPAddress:   "", // not started yet; set in startContainer*Orch
	}
}

func (cnt *AgentContainer) SetIPAddress(ip string) {
	cnt.IPAddress = ip
}
