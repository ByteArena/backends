package types

import (
	"encoding/json"
	"net"

	"github.com/bytearena/ecs"
	uuid "github.com/satori/go.uuid"
)

type _privateAgentMessageMethod string

func (p _privateAgentMessageMethod) String() string {
	return string(p)
}

var AgentMessageType = struct {
	Handshake _privateAgentMessageMethod
	Mutation  _privateAgentMessageMethod
}{
	Handshake: _privateAgentMessageMethod("Handshake"),
	Mutation:  _privateAgentMessageMethod("Mutation"),
}

///////////////////////////////////////////////////////////////////////////////
// The message wrapper; holds a Payload
///////////////////////////////////////////////////////////////////////////////
type AgentMessage struct {
	AgentId     uuid.UUID                  `json:"agentid"`
	Method      _privateAgentMessageMethod `json:"method"`
	Payload     json.RawMessage            `json:"payload"`
	EmitterConn net.Conn
}

func (m AgentMessage) GetAgentId() uuid.UUID {
	return m.AgentId
}

func (m AgentMessage) GetMethod() _privateAgentMessageMethod {
	return m.Method
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
