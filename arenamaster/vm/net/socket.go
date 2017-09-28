package net

import (
	// "bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/bytearena/bytearena/common/utils"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/net/ipv4"
)

const (
	BUFFERSIZE = 1500
)

var remoteFd, _ = net.ListenPacket("ip4:tcp", "0.0.0.0")

func ListenSocket(addr string) {
	listenAddr, listenAddrErr := net.ResolveTCPAddr("tcp4", addr)
	utils.Check(listenAddrErr, "Could not resolve tcp addr")

	lstnConn, err := net.ListenTCP("tcp", listenAddr)
	if nil != err {
		log.Fatalln("Unable to listen on TCP socket:", err)
	}

	defer lstnConn.Close()

	for {
		// Listen for an incoming connection.
		conn, err := lstnConn.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			panic("")
		}

		// Handle connections in a new goroutine.
		go handleRequest(conn)
	}

	// recv in separate thread
	// go func() {
	// 	buf := make([]byte, BUFFERSIZE)
	// 	for {
	// 		n, addr, err := lstnConn.ReadFromUDP(buf)
	// 		// just debug
	// 		header, _ := ipv4.ParseHeader(buf[:n])
	// 		fmt.Printf("Received %d bytes from %v: %+v\n", n, addr, header)
	// 		if err != nil || n == 0 {
	// 			fmt.Println("Error: ", err)
	// 			continue
	// 		}
	// 		// write to TUN interface
	// 		// ifce.Write(buf[:n])
	// 		log.Println(buf[:n])
	// 	}
	// }()

	// // and one more loop
	// packet := make([]byte, BUFFERSIZE)
	// for {
	// 	plen, err := ifce.Read(packet)
	// 	if err != nil {
	// 		break
	// 	}
	// 	// debug :)
	// 	header, _ := ipv4.ParseHeader(packet[:plen])
	// 	fmt.Printf("Sending to remote: %+v (%+v)\n", header, err)
	// 	// real send
	// 	lstnConn.WriteToUDP(packet[:plen], remoteAddr)
	// }

	// lstnConn.Close()
}

// Handles incoming requests.
func handleRequest(conn net.Conn) {
	defer conn.Close()

	packet := make([]byte, BUFFERSIZE)
	for {
		plen, err := conn.Read(packet)
		if err != nil {
			log.Println("Cannot read: " + err.Error())
			break
		}

		header, _ := ipv4.ParseHeader(packet[:plen])
		fmt.Printf("- - - - - - - - - - Received packet: %+v\n", header)
		// log.Println("packet----------------------------------------------------------------------------------------------------", packet[:plen])

		// reader := bufio.NewReader(conn)
		// buf, err := reader.ReadBytes('\n')
		// if err != nil {
		// 	log.Println("Connexion closed unexpectedly; " + err.Error())
		// 	return
		// }

		// log.Println(buf)

		// Decode a packet
		// packet := gopacket.NewPacket(packet[:plen], layers.LayerTypeTCP, gopacket.Default)
		packet := gopacket.NewPacket(packet[:plen], layers.LayerTypeEthernet, gopacket.Default)

		fmt.Printf("%s\n", packet.Dump())

		// Get the TCP layer from this packet
		if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			// Get actual TCP data from this layer
			tcp, _ := tcpLayer.(*layers.TCP)
			fmt.Printf("From src port %d to dst port %d\n", tcp.SrcPort, tcp.DstPort)

			// remoteFd.WriteTo(, &net.IPAddr{IP: tcp.}))
		}

		if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
			// Get actual TCP data from this layer
			panic("IP")
			// remoteFd.WriteTo(, &net.IPAddr{IP: tcp.}))
		}

	}

}

func to4byte(addr string) [4]byte {
	parts := strings.Split(addr, ".")
	b0, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Fatalf("to4byte: %s (latency works with IPv4 addresses only, but not IPv6!)\n", err)
	}
	b1, _ := strconv.Atoi(parts[1])
	b2, _ := strconv.Atoi(parts[2])
	b3, _ := strconv.Atoi(parts[3])
	return [4]byte{byte(b0), byte(b1), byte(b2), byte(b3)}
}
