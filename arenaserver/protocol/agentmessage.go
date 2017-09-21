package protocol

import (
	"encoding/json"
	"net"

	uuid "github.com/satori/go.uuid"
)

type AgentMessage struct {
	AgentId     uuid.UUID
	Type        string
	Payload     json.RawMessage
	EmitterConn net.Conn
}

func (m AgentMessage) GetAgentId() uuid.UUID {
	return m.AgentId
}

func (m AgentMessage) GetType() string {
	return m.Type
}

func (m AgentMessage) GetPayload() json.RawMessage {
	return m.Payload
}

func (m AgentMessage) GetEmitterConn() net.Conn {
	return m.EmitterConn
}
