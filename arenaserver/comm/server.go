package comm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/bytearena/bytearena/arenaserver/types"

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
								NewFromString("Agent connection has been aborted").
								SetContext("timeout", CONNECTION_TO_MESSAGE_DEADLINE.String()).
								With(bettererrors.NewFromString("Handshake timeout"))

							// Avoid crashes when agent crashes Issue #108
							s.Log(EventConnDisconnected{
								Err:  berror,
								Conn: conn,
							})

							break
						}

					case buf := <-dataChan:
						{
							log.Println("got data")

							// Cancel deadline
							gotData = true

							// Unmarshal message (unwrapping in an AgentMessage structure)
							var msg types.AgentMessage
							err = json.Unmarshal(buf, &msg)
							if err != nil {
								berror := bettererrors.
									NewFromString("Failed to unmarshal incoming JSON in CommServer::Listen()").
									With(bettererrors.NewFromErr(err)).
									SetContext("buff", string(buf))

								s.Log(EventWarn{berror})
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

					case err := <-errorChan:
						{
							// Cancel deadline
							gotData = true

							berror := bettererrors.
								NewFromString("Connexion closed unexpectedly").
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
