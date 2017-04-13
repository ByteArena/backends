package protocol

type AgentCommunicator interface {
	NetSender
	StateMutationPusher
}
