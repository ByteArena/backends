package comm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/bytearena/bytearena/arenaserver/protocol"
	"github.com/bytearena/bytearena/common/utils"
)

type CommDispatcher interface {
	DispatchAgentMessage(msg protocol.MessageWrapper)
}

type CommServer struct {
	address  string
	listener net.Listener
}

// Creates new tcp server instance
func NewCommServer(address string) *CommServer {
	return &CommServer{
		address: address,
	}
}

func (s *CommServer) Send(message []byte, conn net.Conn) error {

	_, err := conn.Write(message)
	if err != nil {
		return err
	}

	return nil
}

func (s *CommServer) Listen(dispatcher CommDispatcher) error {

	ln, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("Comm server could not listen on %s; %s", s.address, err.Error())
	}

	s.listener = ln
	defer s.listener.Close()
	for {

		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}

		conn.SetReadDeadline(time.Now().Add(time.Second * 10))

		go func() {
			for {
				reader := bufio.NewReader(conn)
				buf, err := reader.ReadBytes('\n')
				if err != nil {
					log.Panicln(err)
					return
				}

				// no more read deadline
				conn.SetReadDeadline(time.Time{})

				// Unmarshal message (unwrapping in an AgentMessage structure)
				var msg protocol.MessageWrapperImp
				err = json.Unmarshal(buf, &msg)
				utils.Check(err, "Failed to unmarshal incoming JSON in CommServer::Listen();"+string(buf))

				msg.EmitterConn = conn

				go dispatcher.DispatchAgentMessage(msg)
			}
		}()
	}

	return nil
}
