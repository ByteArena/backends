package types

import (
	"encoding/json"
	"net"

	"github.com/bytearena/ecs"
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
	AgentId     uuid.UUID                `json:"agentid"`
	Type        _privateAgentMessageType `json:"type"`
	Payload     json.RawMessage          `json:"payload"`
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
// Protocol versions
///////////////////////////////////////////////////////////////////////////////
var (
	PROTOCOL_VERSION_CLEAR_BETA = "clear_beta"
	PROTOCOL_VERSION_CLEAR_V1   = "clear_v1"

	PROTOCOL_VERSIONS = []string{
		PROTOCOL_VERSION_CLEAR_BETA,
		PROTOCOL_VERSION_CLEAR_V1,
	}
)

///////////////////////////////////////////////////////////////////////////////
// Handshake payload
///////////////////////////////////////////////////////////////////////////////
type AgentMessagePayloadHandshake struct {
	Version string `json:"version"`
}

///////////////////////////////////////////////////////////////////////////////
// Mutation payload
///////////////////////////////////////////////////////////////////////////////
type AgentMessagePayloadMutation struct {
	Method    string          `json:"method"`
	Arguments json.RawMessage `json:"arguments"`
}

func (m AgentMessagePayloadMutation) GetMethod() string {
	return m.Method
}

func (m AgentMessagePayloadMutation) GetArguments() json.RawMessage {
	return m.Arguments
}

type AgentMutationBatch struct {
	AgentProxyUUID uuid.UUID
	AgentEntityId  ecs.EntityID
	Mutations      []AgentMessagePayloadMutation `json:"mutations"`
}

type AgentMutationBatcherInterface interface {
	PushMutationBatch(batch AgentMutationBatch)
}
