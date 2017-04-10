package protocol

// Interface to avoid circular dependencies between server and agent

type AgentCommOperator interface {
	GetNetworkCommServer() NetSender
}
