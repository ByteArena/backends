package vm

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"

	"golang.org/x/net/ipv4"

	// "github.com/rkt/rkt/networking/tuntap"
	// "github.com/milosgajdos83/tenus"
	"github.com/rkt/rkt/networking/tuntap"
	"github.com/songgao/water"
	"github.com/vishvananda/netlink"
)

const (
	// I use TUN interface, so only plain IP packet, no ethernet header + mtu is set to 1300
	BUFFERSIZE = 1500
	MTU        = 1300
	SUBNET     = "10.1.0.10/24"
	REMOTEIP   = "192.168.1.17"
	PORT       = 4321
)

func decodeHex(src []byte) string {
	dst := make([]byte, hex.DecodedLen(len(src)))
	n, err := hex.Decode(dst, src)

	if err != nil {
		return err.Error()
	}

	return string(dst[:n])
}

func createTapInterface() (*water.Interface, error) {
	// config := water.Config{
	// 	DeviceType: water.TUN,
	// }

	ifName, persistErr := tuntap.CreatePersistentIface("ba-tap%d", tuntap.Tun)

	if persistErr != nil {
		panic(persistErr)
	}

	ifce, err := water.NewTUN(ifName)
	if err != nil {
		return nil, err
	}

	log.Println("Interface allocated:", ifce.Name())

	return ifce, nil
}

func createTapLink(ifce *water.Interface) error {
	// lIP, lNet, err := net.ParseCIDR(SUBNET)
	// device := ifce.Name()

	// runIP("link", "set", "dev", device, "mtu", MTU)
	// runIP("addr", "add", SUBNET, "dev", device)
	// runIP("link", "set", "dev", device, "up")

	// link, err := tenus.NewLinkFrom(ifce.Name())
	// if nil != err {
	// 	log.Fatalln("Unable to get interface info", err)
	// }

	// err = link.SetLinkMTU(MTU)
	// if nil != err {
	// 	log.Fatalln("Unable to set MTU to 1300 on interface")
	// }

	// err = link.SetLinkIp(lIP, lNet)
	// if nil != err {
	// 	log.Fatalln("Unable to set IP to ", lIP, "/", lNet, " on interface")
	// }

	// err = link.SetLinkUp()
	// if nil != err {
	// 	log.Fatalln("Unable to UP interface")
	// }
	ifName := ifce.Name()

	link, err := netlink.LinkByName(ifName)
	if err != nil {
		panic(fmt.Sprintf("cannot find link %q", ifName))
	}

	if err := netlink.LinkSetUp(link); err != nil {
		panic(fmt.Sprintf("cannot set link up %q", ifName))
	}

	br := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: "docker0",
			MTU:  MTU,
		},
	}

	if err := netlink.LinkSetUp(br); err != nil {
		panic(err)
	}

	// err = netlink.LinkSetMaster(link, br)
	// if err != nil {
	// 	panic(err)
	// }

	return nil
}

func listenTap(ifce *water.Interface) {
	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", REMOTEIP, PORT))
	if nil != err {
		log.Fatalln("Unable to resolve remote addr:", err)
	}
	// listen to local socket
	lstnAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%v", PORT))
	if nil != err {
		log.Fatalln("Unable to get UDP socket:", err)
	}
	lstnConn, err := net.ListenUDP("udp", lstnAddr)
	if nil != err {
		log.Fatalln("Unable to listen on UDP socket:", err)
	}

	// recv in separate thread
	go func() {
		buf := make([]byte, BUFFERSIZE)
		for {
			n, addr, err := lstnConn.ReadFromUDP(buf)
			// just debug
			header, _ := ipv4.ParseHeader(buf[:n])
			fmt.Printf("Received %d bytes from %v: %+v\n", n, addr, header)
			if err != nil || n == 0 {
				fmt.Println("Error: ", err)
				continue
			}
			// write to TUN interface
			ifce.Write(buf[:n])
		}
	}()

	// and one more loop
	packet := make([]byte, BUFFERSIZE)
	for {
		plen, err := ifce.Read(packet)
		if err != nil {
			break
		}
		// debug :)
		header, _ := ipv4.ParseHeader(packet[:plen])
		fmt.Printf("Sending to remote: %+v (%+v)\n", header, err)
		// real send
		lstnConn.WriteToUDP(packet[:plen], remoteAddr)
	}

	lstnConn.Close()
}

func runIP(args ...string) {
	cmd := exec.Command("/sbin/ip", args...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	err := cmd.Run()

	if err != nil {
		log.Println("Error running /sbin/ip:", err)
	}
}
