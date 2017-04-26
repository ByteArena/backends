package comm

import (
	"net"

	"github.com/netgusto/bytearena/server/protocol"

	"encoding/json"

	"github.com/netgusto/bytearena/utils"
)

// Client holds info about connection
type CommClient struct {
	udpaddr net.Addr
	Server  *CommServer
}

type CommDispatcher interface {
	DispatchAgentMessage(msg protocol.MessageWrapper)
}

// UDP server
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
	utils.Check(err, "Cannot listen on "+s.address)

	s.conn = listener

	defer s.conn.Close()
	for {
		buf := make([]byte, s.buffsize)
		n, addr, err := s.conn.ReadFrom(buf)
		utils.Check(err, "Read from buffer failed in CommServer::Listen()")

		// Unmarshal message (unwrapping in an AgentMessage structure)
		var msg protocol.MessageWrapperImp
		err = json.Unmarshal(buf[0:n], &msg)
		utils.Check(err, "Failed to unmarshal incoming JSON in CommServer::Listen()")

		msg.EmitterAddr = addr

		go dispatcher.DispatchAgentMessage(msg)
	}
}
