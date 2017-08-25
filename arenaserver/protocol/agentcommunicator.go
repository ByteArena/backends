package protocol

type AgentCommunicatorInterface interface {
	NetSenderInterface
	StateMutationPusherInterface
}
