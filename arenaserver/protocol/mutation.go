package protocol

import (
	"encoding/json"

	uuid "github.com/satori/go.uuid"
)

type MutationMessage struct {
	Method    string
	Arguments json.RawMessage
}

func (m MutationMessage) GetMethod() string {
	return m.Method
}

func (m MutationMessage) GetArguments() json.RawMessage {
	return m.Arguments
}

type StateMutationBatch struct {
	AgentId   uuid.UUID
	Mutations []MutationMessage
}

type StateMutationPusherInterface interface {
	PushMutationBatch(batch StateMutationBatch)
}
