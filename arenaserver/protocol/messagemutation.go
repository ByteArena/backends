package protocol

import "encoding/json"

type MessageMutations interface {
	GetMutations() []MessageMutation
}

type MessageMutation interface {
	GetMethod() string
	GetArguments() json.RawMessage
}

type MessageMutationsImp struct {
	Mutations []MessageMutationImp
}

type MessageMutationImp struct {
	Method    string
	Arguments json.RawMessage
}

func (m MessageMutationsImp) GetMutations() []MessageMutationImp {
	return m.Mutations
}

func (m MessageMutationImp) GetMethod() string {
	return m.Method
}

func (m MessageMutationImp) GetArguments() json.RawMessage {
	return m.Arguments
}
