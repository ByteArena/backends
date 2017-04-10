package protocol

import "net"

type NetSender interface {
	Send(message []byte, addr net.Addr)
}
