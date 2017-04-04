package server

import (
	"bufio"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/netgusto/bytearena/server/state"
	uuid "github.com/satori/go.uuid"
	"github.com/ttacon/chalk"

	"encoding/json"
)

type RPCHandshakeRequest struct {
	Agent     string
	Handshake string
}

type RPCRequest struct {
	Method    string
	RequestId uint32
	Arguments []interface{}
}

type RPCResponse struct {
	RequestId uint32
	Results   []interface{}
}

// Client holds info about connection
type TCPClient struct {
	conn              net.Conn
	Server            *TCPServer
	agent             *Agent
	hastickedgoodturn chan state.Tickturn
	//hastickedbadturn  chan bool
	//hastimedoutfortick chan bool
	//lastturn           state.Tickturn // dernier turn soumis (ou en timeout)
}

// TCP server
type TCPServer struct {
	Clients []*TCPClient
	address string // Address to open connection: localhost:9999
	proto   string
	swarm   *Swarm
	//state.Tickturnopen bool
	//late int
	mutex        *sync.Mutex
	expectedturn state.Tickturn
}

// Read client data from channel
func (c *TCPClient) listen() {
	reader := bufio.NewReader(c.conn)
	for {
		buf, err := reader.ReadBytes('\n')
		if err != nil {
			c.conn.Close()
			c.Server.swarm.OnClientConnectionClosed(c, err)
			defer func() {
				c.Server.removeClient(c)
			}()
			return
		}
		c.Server.OnNewMessage(c, buf)
	}
}

// Send message to client
func (c *TCPClient) Send(message []byte) error {
	writer := bufio.NewWriter(c.conn)
	_, err := writer.Write(message)
	if err != nil {
		return err
	}
	return writer.Flush()

}

func (c *TCPClient) Conn() net.Conn {
	return c.conn
}

func (c *TCPClient) Close() error {
	return c.conn.Close()
}

func (s *TCPServer) SetExpectedTurn(turn state.Tickturn) {
	s.mutex.Lock()
	s.expectedturn = turn
	s.mutex.Unlock()
}

func (s *TCPServer) GetExpectedTurn() state.Tickturn {
	s.mutex.Lock()
	res := s.expectedturn
	s.mutex.Unlock()
	return res
}

func (s *TCPServer) OnNewMessage(c *TCPClient, message []byte) {

	go func(cli *TCPClient, srv *TCPServer) {

		expectedturn := s.GetExpectedTurn()

		var request RPCRequest
		err := json.Unmarshal(message, &request)
		if err != nil {
			log.Panicln(err)
		}

		//var args []interface{}

		if request.Method == "mutations" {
			/*if !s.state.Tickturnopen {
				return
			}*/

			if len(request.Arguments) == 0 {
				log.Println("MISSING TICK TURN NUMBER !!")
				return
			}

			turnedtick, ok := request.Arguments[0].(float64)
			if !ok {
				log.Println("INVALID TICK TURN NUMBER !!")
				return
			}

			turnedtickint := uint32(turnedtick)

			// this client has ticked, it won't timeout
			// Make sure there's a consumer side, otherwise this gofunc will be blocked here

			if turnedtickint != expectedturn.seq {

				//cli.hastickedbadturn <- true

				log.Print(chalk.Red)
				log.Println("LATE FRAME !! from tick " + strconv.Itoa(int(turnedtickint)) + "; expected " + srv.expectedturn.String())
				log.Print(chalk.Reset)

				// This tick batch is late; it won't be registered
				return
			}

			mutationbatch := StateMutationBatch{
				Turn:  expectedturn,
				Agent: cli.agent,
			}

			genericmutations := request.Arguments[1].([]interface{})
			for _, genericmutation := range genericmutations {

				args, ok := genericmutation.([]interface{})
				if !ok {
					log.Println("NOPE")
				}

				method := args[0].(string)

				mutationbatch.Mutations = append(mutationbatch.Mutations, StateMutation{
					action:    method,
					arguments: args[1:],
				})
			}

			srv.swarm.PushMutationBatch(mutationbatch)

			cli.hastickedgoodturn <- expectedturn
			return

		}

		//log.Println("LATE FRAMES", s.late)

		procresult, err := srv.swarm.OnProcedureCall(cli, request.Method, request.Arguments)
		if err != nil {
			log.Panicln(err)
		}

		var response RPCResponse
		response.RequestId = request.RequestId
		response.Results = append(response.Results, procresult)

		buf, err := json.Marshal(response)
		if err != nil {
			log.Panicln(err)
		}

		buf = append(buf, '\n')
		c.Send(buf)
	}(c, s)
}

// Start network server
func (s *TCPServer) Listen() error {
	listener, err := net.Listen(s.proto, s.address)
	if err != nil {
		log.Panicln(err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Panicln(err)
		}

		// Waiting for handshake
		reader := bufio.NewReader(conn)
		buf, err := reader.ReadBytes('\n') // TODO: handle some timeout here, if handshake never achieved
		if err != nil {
			log.Println(err)
		}

		// Unmarshal handshake request
		var handshake RPCHandshakeRequest
		err = json.Unmarshal(buf, &handshake)
		if err != nil {
			log.Panicln(err) // TODO: handle client rejection if handshake failed
		}

		if handshake.Agent == "" {
			log.Panicln("Handshake with empty agentid !")
		}

		agent, err := s.swarm.FindAgent(handshake.Agent)
		if err != nil {
			log.Panicln(err)
		}

		if agent == nil {
			log.Panicln("Handshake : agentid does not match any known agent !")
		}

		// Handshake successful ! Matching agent is found and bound to TCPClient
		log.Println("Received handshake from agent " + handshake.Agent)

		//conn.SetDeadline(t)
		client := &TCPClient{
			agent:             agent,
			conn:              conn,
			Server:            s,
			hastickedgoodturn: make(chan state.Tickturn, 10), // can buffer up to 10 turns, to avoid blocking
			//hastickedbadturn:  make(chan bool),         // can buffer up to 10 turns, to avoid blocking
		}
		agent.tcp = client

		go client.listen()

		s.Clients = append(s.Clients, client)
		s.swarm.OnNewClient(client)

		if len(s.Clients) == s.swarm.nbexpectedagents { // Clients et pas swarm.agents, car Client représente les agents effectivement connectés
			s.swarm.OnAgentsReady()
		}
	}
}

func (s *TCPServer) Broadcast(message []byte) {
	for _, client := range s.Clients[:] {
		go func(cli *TCPClient) {
			_ = cli.Send(message)
		}(client)
	}
}

// Creates new tcp server instance
func NewTCPServer(proto, address string, swarm *Swarm) *TCPServer {
	server := &TCPServer{
		address: address,
		proto:   proto,
		swarm:   swarm,
		mutex:   &sync.Mutex{},
		//state.Tickturnopen: false,
	}

	return server
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return true // completed normally
	case <-time.After(timeout):
		return false // timed out
	}
}

func chanTimeout(ch chan state.Tickturn, timeout time.Duration) bool {
	select {
	case <-ch:
		return true // completed normally
	case <-time.After(timeout):
		return false // timed out
	}
}

func (server *TCPServer) StartTicking(tickduration time.Duration, stopticking chan bool, ontick func(took time.Duration)) {

	go func() {

		var turn state.Tickturn
		log.Println("Start ticking")

		timeoutduration := tickduration * 60 / 100
		ticker := time.Tick(tickduration)

		for {
			select {
			case <-stopticking:
				{
					log.Println("Stop Ticking !", turn)
					return
				}
			case <-ticker:
				{
					start := time.Now()

					turn = turn.Next()
					server.SetExpectedTurn(turn)

					log.Println("Tick !", turn)

					// on met à jour le swarm
					server.swarm.update(turn)

					// On ticke chaque client
					for _, client := range server.Clients[:] {
						go func(client *TCPClient, turn state.Tickturn, perception Perception) {
							perceptionjson, _ := json.Marshal(perception)
							message := []byte("{\"Method\": \"tick\", \"Arguments\": [" + strconv.Itoa(int(turn.seq)) + "," + string(perceptionjson) + "]}\n")
							client.Send(message)
						}(client, turn, client.agent.GetPerception())
					}

					// On attend la réponse de chaque client, jusqu'au timeout
					wg := &sync.WaitGroup{}
					wg.Add(len(server.Clients))
					for _, client := range server.Clients[:] {
						go func(client *TCPClient) {
							if chanTimeout(client.hastickedgoodturn, timeoutduration) {
								//log.Print(chalk.Green)
								//log.Println("ALL CLIENTS ON TIME", chalk.Reset)
							} else {
								//log.Print(chalk.Magenta)
								//log.Println("SOME CLIENTS TIMED OUT", chalk.Reset)
							}

							wg.Done()
						}(client)
					}

					wg.Wait()

					// For a reason yet to be determined, this is required, otherwised mutations might be processed too early
					time.Sleep(time.Millisecond)

					ontick(time.Now().Sub(start))
				}
			}
		}
	}()
}

func (s *TCPServer) removeClient(c *TCPClient) {
	log.Println("Removing client !!!")

	// TODO: thread-safe process this operation
	found := -1
	// on trouve l'index du client dans le tableau
	for i, client := range s.Clients[:] {
		if client == c {
			found = i
			break
		}
	}

	if found > -1 {
		s.Clients[len(s.Clients)-1], s.Clients[found] = s.Clients[found], s.Clients[len(s.Clients)-1]
		s.Clients = s.Clients[:len(s.Clients)-1]
	}
}
