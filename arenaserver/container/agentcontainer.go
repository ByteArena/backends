package container

import (
	"io"
	"os"

	uuid "github.com/satori/go.uuid"

	"github.com/bytearena/bytearena/arenaserver/types"
)

type AgentContainer struct {
	AgentId     uuid.UUID
	containerid types.ContainerId
	ImageName   string
	IPAddress   string

	LogReader io.ReadCloser
	LogWriter *os.File
}

func NewAgentContainer(agentid uuid.UUID, containerid types.ContainerId, imageName string) *AgentContainer {
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

func (cnt *AgentContainer) SetLogger(reader io.ReadCloser, writer *os.File) {
	cnt.LogReader = reader
	cnt.LogWriter = writer
}
