package server

import (
	"bufio"
	"log"
	"math"
	"net"
	"strconv"
	"sync"
	"time"

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
	conn      net.Conn
	Server    *TCPServer
	agent     *Agent
	hasticked func()
}

// TCP server
type TCPServer struct {
	Clients      []*TCPClient
	address      string // Address to open connection: localhost:9999
	proto        string
	tickturn     uint32
	swarm        *Swarm
	tickturnopen bool
	late         int
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

func (s *TCPServer) OnNewMessage(c *TCPClient, message []byte) {

	go func(cli *TCPClient, srv *TCPServer) {

		var request RPCRequest
		err := json.Unmarshal(message, &request)
		if err != nil {
			log.Panicln(err)
		}

		//var args []interface{}

		if request.Method == "mutations" {
			if !s.tickturnopen {
				return
			}

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

			if turnedtickint != srv.tickturn {
				s.late++
				log.Print(chalk.Red)
				log.Println("LATE FRAME !! from tick " + strconv.Itoa(int(turnedtickint)) + "; expected " + strconv.Itoa(int(srv.tickturn)))
				log.Print(chalk.Reset)
				return
			}

			// this client has ticked, it won't timeout
			cli.hasticked()

			mutationbatch := &StateMutationBatch{
				Turn:  turnedtickint,
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
			agent:     agent,
			conn:      conn,
			Server:    s,
			hasticked: func() {},
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
func TCPServerNew(proto, address string, swarm *Swarm) *TCPServer {
	server := &TCPServer{
		address:      address,
		proto:        proto,
		swarm:        swarm,
		tickturnopen: false,
	}

	return server
}

func toFixed(val float64, places int) (newVal float64) {
	roundOn := 0.5
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func (s *TCPServer) StartTicking(tickduration time.Duration, stopticking chan bool, ontick func(allticked bool, took time.Duration)) {

	ticktimeout := tickduration * 90 / 100
	s.tickturn = 0
	ticker := time.Tick(tickduration)

	//lasttick := time.Now().UnixNano()

	//blueOnWhite := chalk.Black.NewStyle().WithBackground(chalk.White)

	go func(server *TCPServer) {
		for {
			select {
			case <-stopticking:
				{
					return
				}
			case <-ticker:
				{
					// TODO: gestion concurrente améliorée (pas de tickturnopen muable partagé notamment)
					// et amélioration du waitgroup, et de la cloture des coroutines restant en vol après le timeout

					nbticked := 0
					server.tickturn++
					server.tickturnopen = true
					// à chaque tick

					/*
						log.Print(blueOnWhite)
						log.Println("Beginning tick turn "+strconv.Itoa(int(server.tickturn)), chalk.Reset)
						log.Println("")
					*/

					// On crée un timeout
					timeout := time.NewTimer(ticktimeout)

					wg := sync.WaitGroup{}
					hasticked := func() {
						if !server.tickturnopen {
							return
						}
						wg.Done()
						nbticked++
					}

					// Update swarm state
					// TODO: handle this after the tick, not before ?
					server.swarm.update()

					for _, client := range server.Clients[:] {
						wg.Add(1)
						client.hasticked = hasticked
						perception := client.agent.GetPerception()
						//log.Println(perception)
						perceptionjson, _ := json.Marshal(perception)
						//log.Println(string(perceptionjson))
						message := []byte("{\"Method\": \"tick\", \"Arguments\": [" + strconv.Itoa(int(server.tickturn)) + "," + string(perceptionjson) + "]}\n")
						client.Send(message)
					}

					start := time.Now()
					//server.Broadcast([]byte("{\"Method\": \"tick\", \"Arguments\": [" + strconv.Itoa(int(server.tickturn)) + "]}\n"))

					allticked := make(chan bool)
					go func(wg *sync.WaitGroup) {
						wg.Wait()
						allticked <- true
					}(&wg)

					haveallticked := false

					select {
					case <-timeout.C:
						server.tickturnopen = false
						log.Print(chalk.Red)
						log.Print("Timed out (", len(s.Clients)-nbticked, " clients timed out)")
						log.Print(chalk.Reset)

						log.Println(nbticked)
						for i := 0; i < len(s.Clients)-nbticked; i++ {
							//log.Println("So far so good")
							wg.Done()
						}

						break
					case <-allticked:
						timeout.Stop()
						server.tickturnopen = false
						haveallticked = true
						log.Println("All agents ticked for the turn")
						break
					}

					ontick(haveallticked, time.Now().Sub(start))

				}
			}
		}
	}(s)
}

func (s *TCPServer) removeClient(c *TCPClient) {
	log.Println("Removing client !!!")

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
