package protocol

type AgentCommunicatorInterface interface {
	NetSenderInterface
	AgentMutationBatcherInterface
}
