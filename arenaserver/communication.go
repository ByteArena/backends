package arenaserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/arenaserver/agent"
	"github.com/bytearena/bytearena/arenaserver/comm"
	arenaservertypes "github.com/bytearena/bytearena/arenaserver/types"
	"github.com/bytearena/bytearena/common/utils"
	pkgerrors "github.com/pkg/errors"

	bettererrors "github.com/xtuc/better-errors"
)

var (
	LISTEN_ADDR = net.IP{0, 0, 0, 0}
)

func (s *Server) listen() chan interface{} {
	serveraddress := LISTEN_ADDR.String() + ":" + strconv.Itoa(s.port)
	s.commserver = comm.NewCommServer(serveraddress)

	// Consum com server events
	go func() {
		for {
			msg := <-s.commserver.Events()

			if s.gameIsRunning == false {
				return // ignore message
			}

			switch t := msg.(type) {
			case comm.EventLog:
				s.Log(EventLog{t.Value})

			case comm.EventWarn:
				s.Log(EventWarn{t.Err})

			case comm.EventError:
				s.Log(EventError{t.Err})

			// An agent has probaly been disconnected
			// We need to remove it from our state
			case comm.EventConnDisconnected:
				s.clearAgentConn(t.Conn)
				s.Log(EventWarn{t.Err})
				s.ensureEnoughAgentsAreInGame()

			default:
				msg := fmt.Sprintf("Unsupported message of type %s", reflect.TypeOf(msg))
				panic(msg)
			}

		}

	}()

	s.events <- EventLog{"Server listening on port " + strconv.Itoa(s.port)}

	err := s.commserver.Listen(s)
	utils.Check(err, "Failed to listen on "+serveraddress)

	block := make(chan interface{})
	notify.Start("app:stopticking", block)

	return block
}

func (server *Server) ensureEnoughAgentsAreInGame() {
	left := server.nbhandshaked - len(server.agentproxies)
	pourcentLeft := left * 100 / server.nbhandshaked

	if pourcentLeft > POURCENT_LEFT_BEFORE_QUIT {
		server.Log(EventDebug{"Stopping because not enough agents are left"})
		server.Stop()
	}
}

func (server *Server) clearAgentConn(conn net.Conn) {
	server.agentproxiesmutex.Lock()

	for k, agentproxy := range server.agentproxies {
		netAgent, ok := agentproxy.(agent.AgentProxyNetworkInterface)

		if ok && netAgent.GetConn() == conn {

			// Remove agent from our state
			delete(server.agentproxies, k)
			delete(server.agentimages, k)
			delete(server.agentproxieshandshakes, k)

			server.Log(EventDebug{fmt.Sprintf("Removing %s from state", netAgent.String())})

			break
		}

	}

	server.agentproxiesmutex.Unlock()
}

/* <implementing types.AgentCommunicatorInterface> */
func (s *Server) NetSend(message []byte, conn net.Conn) error {
	return s.commserver.Send(message, conn)
}

func (s *Server) PushMutationBatch(batch arenaservertypes.AgentMutationBatch) {
	s.mutationsmutex.Lock()
	s.pendingmutations = append(s.pendingmutations, batch)
	s.mutationsmutex.Unlock()
}

/* </implementing types.AgentCommunicatorInterface> */

/* <implementing types.CommunicatorDispatcherInterface> */
func (s *Server) ImplementsCommDispatcherInterface() {}
func (s *Server) DispatchAgentMessage(msg arenaservertypes.AgentMessage) error {

	agentproxy, err := s.getAgentProxy(msg.GetAgentId().String())
	if err != nil {
		return errors.New("DispatchAgentMessage: agentid does not match any known agent in received agent message !;" + msg.GetAgentId().String())
	}

	// proto := msg.GetEmitterConn().LocalAddr().Network()
	// ip := strings.Split(msg.GetEmitterConn().RemoteAddr().String(), ":")[0]
	// if proto != "tcp" || ip != "TODO(jerome):take from agent container struct"
	// Problem here: cannot check ip against the one we get from Docker by inspecting the container
	// as the two addresses do not match

	switch msg.GetType() {
	case arenaservertypes.AgentMessageType.Handshake:
		{
			if _, found := s.agentproxieshandshakes[msg.GetAgentId()]; found {
				return errors.New("ERROR: Received duplicate handshake from agent " + agentproxy.String())
			}

			s.agentproxieshandshakes[msg.GetAgentId()] = struct{}{}

			var handshake arenaservertypes.AgentMessagePayloadHandshake
			err = json.Unmarshal(msg.GetPayload(), &handshake)
			if err != nil {
				return pkgerrors.Wrapf(err, "DispatchAgentMessage: Failed to unmarshal agent's (%s) handshake", msg.GetAgentId().String())
			}

			ag, ok := agentproxy.(agent.AgentProxyNetworkInterface)
			if !ok {
				return errors.New("DispatchAgentMessage: Failed to cast agent to NetAgent during handshake for " + ag.String())
			}

			// Check if the agent uses a protocol version we know
			if handshake.Version == "" {
				handshake.Version = "UNKNOWN"
			}

			if !utils.IsStringInArray(arenaservertypes.PROTOCOL_VERSIONS, handshake.Version) {
				return bettererrors.
					NewFromString("Unsupported agent protocol").
					SetContext("agent", ag.String()).
					SetContext("protocol version", handshake.Version)
			}

			ag = ag.SetConn(msg.GetEmitterConn())
			s.setAgentProxy(ag)

			s.events <- EventDebug{"Received handshake from agent " + ag.String() + ""}

			s.nbhandshaked++

			if s.nbhandshaked == s.getNbExpectedagents() {
				s.onAgentsReady()
			}

			// TODO(sven|jerome): handle some timeout here if all agents fail to handshake

			break
		}
	case arenaservertypes.AgentMessageType.Mutation:
		{
			var mutations struct {
				Mutations []arenaservertypes.AgentMessagePayloadMutation
			}

			err = json.Unmarshal(msg.GetPayload(), &mutations)
			if err != nil {
				return errors.New("DispatchAgentMessage: Failed to unmarshal JSON agent mutation payload for agent " + agentproxy.String() + "; " + string(msg.GetPayload()))
			}

			mutationbatch := arenaservertypes.AgentMutationBatch{
				AgentProxyUUID: agentproxy.GetProxyUUID(),
				AgentEntityId:  agentproxy.GetEntityId(),
				Mutations:      mutations.Mutations,
			}

			s.PushMutationBatch(mutationbatch)

			break
		}
	default:
		{
			return errors.New("DispatchAgentMessage: Unknown message type" + msg.GetType().String())
		}
	}

	return nil
}

/* </implementing types.CommunicatorDispatcherInterface> */
