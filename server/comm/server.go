package comm

import (
	"log"
	"net"

	"github.com/netgusto/bytearena/server/protocol"

	"encoding/json"
)

// Client holds info about connection
type CommClient struct {
	udpaddr net.Addr
	Server  *CommServer
}

type CommDispatcher interface {
	DispatchAgentMessage(msg protocol.MessageWrapper)
}

// TCP server
type CommServer struct {
	address  string
	conn     net.PacketConn
	buffsize uint32
}

// Creates new tcp server instance
func NewCommServer(address string, buffsize uint32) *CommServer {
	return &CommServer{
		address:  address,
		buffsize: buffsize,
	}
}

func (s *CommServer) Send(message []byte, addr net.Addr) {
	s.conn.WriteTo(message, addr)
}

func (s *CommServer) Listen(dispatcher CommDispatcher) error {

	listener, err := net.ListenPacket("udp4", s.address)
	if err != nil {
		log.Panicln(err)
	}

	s.conn = listener

	defer s.conn.Close()
	for {
		buf := make([]byte, s.buffsize)
		n, addr, err := s.conn.ReadFrom(buf)
		if err != nil {
			log.Panicln(err)
		}

		// Unmarshal message (unwrapping in an AgentMessage structure)
		var msg protocol.MessageWrapperImp
		err = json.Unmarshal(buf[0:n], &msg)
		if err != nil {
			log.Panicln(err)
		}
		msg.EmitterAddr = addr

		go dispatcher.DispatchAgentMessage(msg)

	}
}
