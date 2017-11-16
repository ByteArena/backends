package comm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/bytearena/bytearena/arenaserver/types"
	"github.com/bytearena/bytearena/common/assert"
	uuid "github.com/satori/go.uuid"

	bettererrors "github.com/xtuc/better-errors"
)

const (
	CONNECTION_TO_MESSAGE_DEADLINE = 3 * time.Second
)

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

func readBytesChan(conn net.Conn) (chan []byte, chan error) {
	dataChan := make(chan []byte)
	errChan := make(chan error)

	reader := bufio.NewReader(conn)

	go func() {

		for {
			buf, err := reader.ReadBytes('\n')

			if err != nil {
				errChan <- err
			} else {
				dataChan <- buf
			}
		}
	}()

	return dataChan, errChan
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
				dataChan, errorChan := readBytesChan(conn)
				gotData := false

				for {

					select {
					case <-time.After(CONNECTION_TO_MESSAGE_DEADLINE):
						if !gotData {
							berror := bettererrors.
								New("Agent connection has been aborted").
								SetContext("timeout", CONNECTION_TO_MESSAGE_DEADLINE.String()).
								With(bettererrors.New("Handshake timeout"))

							// Avoid crashes when agent crashes Issue #108
							s.Log(EventConnDisconnected{
								Err:  berror,
								Conn: conn,
							})

							break
						}

					case buf := <-dataChan:
						{
							// Cancel deadline
							gotData = true

							// Dump traffic
							s.Log(EventRawComm{buf})

							// Unmarshal message (unwrapping in an AgentMessage structure)
							var msg types.AgentMessage
							err = json.Unmarshal(buf, &msg)

							if err != nil {
								berror := bettererrors.
									New("Failed to unmarshal incoming JSON in CommServer::Listen()").
									With(bettererrors.NewFromErr(err)).
									SetContext("string", fmt.Sprintf("\"%s\"", buf)).
									SetContext("raw", fmt.Sprintf("%v", buf))

								assert.AssertBE(false, berror)
							} else {
								msg.EmitterConn = conn

								assert.Assert(msg.AgentId != uuid.Nil, "agentid is null")

								go func() {
									err := dispatcher.DispatchAgentMessage(msg)
									if err != nil {
										berror := bettererrors.
											New("Failed to dispatch agent message").
											With(bettererrors.NewFromErr(err))

										s.Log(EventError{berror})
									}
								}()
							}
						}

					case err := <-errorChan:
						{
							// Cancel deadline
							gotData = true

							berror := bettererrors.
								New("Connexion closed unexpectedly").
								With(bettererrors.NewFromErr(err))

							// Avoid crashes when agent crashes Issue #108
							s.Log(EventConnDisconnected{
								Err:  berror,
								Conn: conn,
							})

							conn.Close()
							return
						}
					}

				}

				conn.Close()
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
