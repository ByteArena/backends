package protocol

import (
	uuid "github.com/satori/go.uuid"
)

type StateMutation struct {
	Action    string
	Arguments []interface{}
}

type StateMutationBatch struct {
	AgentId   uuid.UUID
	Mutations []MessageMutation
}

type StateMutationPusher interface {
	PushMutationBatch(batch StateMutationBatch)
}
