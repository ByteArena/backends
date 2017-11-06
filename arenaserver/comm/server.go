package comm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"

	"github.com/bytearena/bytearena/arenaserver/types"

	bettererrors "github.com/xtuc/better-errors"
)

type EventLog struct{ Value string }
type EventError struct{ Err error }

type CommDispatcherInterface interface {
	DispatchAgentMessage(msg types.AgentMessage) error
	ImplementsCommDispatcherInterface()
}

type CommServer struct {
	address  string
	listener net.Listener

	events chan interface{}
}

// Creates new tcp server instance
func NewCommServer(address string) *CommServer {
	return &CommServer{
		address: address,

		events: make(chan interface{}),
	}
}

func (s *CommServer) Send(message []byte, conn net.Conn) error {

	_, err := conn.Write(message)
	if err != nil {
		return err
	}

	return nil
}

func (s *CommServer) Listen(dispatcher CommDispatcherInterface) error {

	ln, err := net.Listen("tcp4", s.address)
	if err != nil {
		return fmt.Errorf("Comm server could not listen on %s; %s", s.address, err.Error())
	}

	s.listener = ln

	go func() {
		defer s.listener.Close()
		for {

			conn, err := s.listener.Accept()
			if err != nil {
				s.Log(EventError{err})
				continue
			}

			go func() {
				defer conn.Close()
				for {

					reader := bufio.NewReader(conn)
					buf, err := reader.ReadBytes('\n')
					if err != nil {
						// Avoid crashes when agent crashes Issue #108
						s.Log(EventLog{"Connexion closed unexpectedly; " + err.Error()})
						return
					}

					// Unmarshal message (unwrapping in an AgentMessage structure)
					var msg types.AgentMessage
					err = json.Unmarshal(buf, &msg)
					if err != nil {
						s.Log(EventLog{"Failed to unmarshal incoming JSON in CommServer::Listen(); " + string(buf) + ";" + err.Error()})
					} else {
						msg.EmitterConn = conn

						go func() {
							err := dispatcher.DispatchAgentMessage(msg)
							if err != nil {
								berror := bettererrors.
									NewFromString("Failed to dispatch agent message").
									With(err)

								s.Log(EventError{berror})
							}
						}()
					}
				}
			}()
		}
	}()

	return nil
}

func (s *CommServer) Log(l interface{}) {
	go func() {
		s.events <- l
	}()
}

func (s *CommServer) Events() chan interface{} {
	return s.events
}
