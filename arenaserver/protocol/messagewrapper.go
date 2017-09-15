package protocol

import (
	"encoding/json"
	"net"

	uuid "github.com/satori/go.uuid"
)

type MessageWrapperInterface interface {
	GetAgentId() uuid.UUID
	GetType() string
	GetPayload() json.RawMessage
	GetEmitterConn() net.Conn
}

type MessageWrapperImp struct {
	AgentId     uuid.UUID
	Type        string
	Payload     json.RawMessage
	EmitterConn net.Conn
}

func (m MessageWrapperImp) GetAgentId() uuid.UUID {
	return m.AgentId
}

func (m MessageWrapperImp) GetType() string {
	return m.Type
}

func (m MessageWrapperImp) GetPayload() json.RawMessage {
	return m.Payload
}

func (m MessageWrapperImp) GetEmitterConn() net.Conn {
	return m.EmitterConn
}
