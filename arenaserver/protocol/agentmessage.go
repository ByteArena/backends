package protocol

import (
	"encoding/json"
	"net"

	uuid "github.com/satori/go.uuid"
)

type _privateAgentMessageType string

func (p _privateAgentMessageType) String() string {
	return string(p)
}

var AgentMessageType = struct {
	Handshake _privateAgentMessageType
	Mutation  _privateAgentMessageType
}{
	Handshake: _privateAgentMessageType("Handshake"),
	Mutation:  _privateAgentMessageType("Mutation"),
}

///////////////////////////////////////////////////////////////////////////////
// The message wrapper; holds a Payload
///////////////////////////////////////////////////////////////////////////////
type AgentMessage struct {
	AgentId     uuid.UUID
	Type        _privateAgentMessageType
	Payload     json.RawMessage
	EmitterConn net.Conn
}

func (m AgentMessage) GetAgentId() uuid.UUID {
	return m.AgentId
}

func (m AgentMessage) GetType() _privateAgentMessageType {
	return m.Type
}

func (m AgentMessage) GetPayload() json.RawMessage {
	return m.Payload
}

func (m AgentMessage) GetEmitterConn() net.Conn {
	return m.EmitterConn
}

///////////////////////////////////////////////////////////////////////////////
// Handshake payload
///////////////////////////////////////////////////////////////////////////////
type AgentMessagePayloadHandshake struct {
	Greetings string
}

func (h AgentMessagePayloadHandshake) GetGreetings() string {
	return h.Greetings
}

///////////////////////////////////////////////////////////////////////////////
// Mutation payload
///////////////////////////////////////////////////////////////////////////////
type AgentMessagePayloadMutation struct {
	Method    string
	Arguments json.RawMessage
}

func (m AgentMessagePayloadMutation) GetMethod() string {
	return m.Method
}

func (m AgentMessagePayloadMutation) GetArguments() json.RawMessage {
	return m.Arguments
}

type AgentMutationBatch struct {
	AgentId   uuid.UUID
	Mutations []AgentMessagePayloadMutation
}

type AgentMutationBatcherInterface interface {
	PushMutationBatch(batch AgentMutationBatch)
}
