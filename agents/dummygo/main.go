package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)

func CheckError(err error) {
	if err != nil {
		log.Println("Error: ", err)
	}
}

func main() {
	log.Println("Hello, World !")

	host, exists := os.LookupEnv("SWARMHOST")
	if !exists {
		log.Panicln("Missing SWARMHOST env variable")
	}

	port, exists := os.LookupEnv("SWARMPORT")
	if !exists {
		log.Panicln("Missing SWARMPORT env variable")
	}

	agentid, exists := os.LookupEnv("AGENTID")
	if !exists {
		log.Panicln("Missing AGENTID env variable")
	}

	ServerAddr, err := net.ResolveUDPAddr("udp", host+":"+port)
	CheckError(err)

	Conn, err := net.DialUDP("udp", nil, ServerAddr)
	CheckError(err)

	defer Conn.Close()

	// Handshake

	buf := []byte("{ \"AgentId\": \"" + agentid + "\", \"Type\": \"Handshake\", \"Payload\": { \"Greetings\": \"Hello from dummygo !\"} }")
	_, err = Conn.Write(buf)
	CheckError(err)

	turn := 0

	for {
		req := make([]byte, 1024)
		_, _, err := Conn.ReadFrom(req)
		CheckError(err)

		fmt.Println(turn)

		res := []byte("{ \"AgentId\": \"" + agentid + "\", \"Type\": \"Mutation\", \"Payload\": { \"Turn\": " + strconv.Itoa(turn) + ", \"Mutations\": [] } }")
		_, err = Conn.Write(res)

		turn++
	}
}
