package protocol

import (
	"encoding/json"

	uuid "github.com/satori/go.uuid"
)

type AgentMutationMessage struct {
	Method    string
	Arguments json.RawMessage
}

func (m AgentMutationMessage) GetMethod() string {
	return m.Method
}

func (m AgentMutationMessage) GetArguments() json.RawMessage {
	return m.Arguments
}

type AgentMutationBatch struct {
	AgentId   uuid.UUID
	Mutations []AgentMutationMessage
}

type AgentMutationBatcherInterface interface {
	PushMutationBatch(batch AgentMutationBatch)
}
