package protocol

import "encoding/json"

type MessageMutations interface {
	//GetTickTurnSeq() int
	GetMutations() []MessageMutation
}

type MessageMutation interface {
	GetMethod() string
	GetArguments() json.RawMessage
}

type MessageMutationsImp struct {
	//Turn      int
	Mutations []MessageMutationImp
}

type MessageMutationImp struct {
	Method    string
	Arguments json.RawMessage
}

func (m MessageMutationsImp) GetMutations() []MessageMutation {
	res := make([]MessageMutation, len(m.Mutations))
	for i, v := range m.Mutations {
		res[i] = MessageMutation(v)
	}
	return res
}

func (m MessageMutationImp) GetMethod() string {
	return m.Method
}

func (m MessageMutationImp) GetArguments() json.RawMessage {
	return m.Arguments
}
