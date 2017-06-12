package protocol

import (
	"encoding/json"
	"net"

	uuid "github.com/satori/go.uuid"
)

type MessageWrapper interface {
	GetAgentId() uuid.UUID
	GetType() string
	GetPayload() json.RawMessage
	GetEmitterAddr() net.Addr
}

type MessageWrapperImp struct {
	AgentId     uuid.UUID
	Type        string
	Payload     json.RawMessage
	EmitterAddr net.Addr
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

func (m MessageWrapperImp) GetEmitterAddr() net.Addr {
	return m.EmitterAddr
}
