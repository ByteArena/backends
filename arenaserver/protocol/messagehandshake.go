package protocol

type MessageHandshakeInterface interface {
	GetGreetings() string
}

type MessageHandshakeImp struct {
	Greetings string
}

func (h MessageHandshakeImp) GetGreetings() string {
	return h.Greetings
}
