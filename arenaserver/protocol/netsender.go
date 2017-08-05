package protocol

import "net"

// Interface to avoid circular dependencies between server and agent

type NetSender interface {
	NetSend(message []byte, conn net.Conn) error
}
