package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/context"

	"github.com/gorilla/websocket"
	"github.com/kardianos/osext"
	"github.com/netgusto/bytearena/server"
)

type vizmessage struct {
	Agents []vizagentmessage
}

type vizagentmessage struct {
	X    float64
	Y    float64
	Kind string
}

type wsincomingmessage struct {
	messageType int
	p           []byte
	err         error
}

type cmdenvironment struct {
	host     string
	port     int
	tps      int
	agents   int
	agentimp string
}

func wsendpoint(w http.ResponseWriter, r *http.Request, stateChan chan server.SwarmState) {

	upgrader := websocket.Upgrader{} // use default options

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	defer func(c *websocket.Conn) {
		c.Close()
		log.Println("Closing !!!")
	}(c)

	clientclosedsocket := make(chan bool)
	c.SetCloseHandler(func(code int, text string) error {
		clientclosedsocket <- true
		return nil
	})

	incomingmsg := make(chan wsincomingmessage)
	go func(client *websocket.Conn, ch chan wsincomingmessage) {
		messageType, p, err := client.ReadMessage()
		ch <- wsincomingmessage{
			messageType,
			p,
			err,
		}
	}(c, incomingmsg)

	for {
		select {
		case <-incomingmsg:
			{
				// just consume msg and allow gorilla to trigger the closehandler when client sends xFFx00 (close signal)
			}
		case <-clientclosedsocket:
			{
				log.Println("<-clientclosedsocket")
				return
			}
		case swarmstate := <-stateChan:
			{
				msg := vizmessage{}

				for _, state := range swarmstate.Agents {
					x, y := state.Position.Get()

					msg.Agents = append(msg.Agents, vizagentmessage{
						X:    x,
						Y:    y,
						Kind: "agent",
					})
				}

				x, y := swarmstate.Pin.Get()

				msg.Agents = append(msg.Agents, vizagentmessage{
					X:    x,
					Y:    y,
					Kind: "attractor",
				})

				json, err := json.Marshal(msg)
				if err != nil {
					log.Println("json error, wtf")
					return
				}

				//log.Println("Tick")
				err = c.WriteMessage(websocket.TextMessage, json)
				if err != nil {
					log.Println("write error !")
					return
				}
			}
		}
	}

}

func visualization(swarm *server.Swarm, host string, port int) {

	basepath := "./client/"

	addr := flag.String("addr", host+":"+strconv.Itoa(port), "http service address")

	stateobserver := swarm.SubscribeStateObservation()
	staterelays := make([]chan server.SwarmState, 0)
	staterelaymutex := &sync.Mutex{}
	go func() {
		for {
			select {
			case curstate := <-stateobserver:
				{
					staterelaymutex.Lock()
					for _, relay := range staterelays {
						relay <- curstate
					}
					staterelaymutex.Unlock()
				}
			}
		}
	}()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		staterelaymutex.Lock()
		relay := make(chan server.SwarmState)
		staterelays = append(staterelays, relay)
		staterelaymutex.Unlock()
		wsendpoint(w, r, relay)
	})

	http.HandleFunc("/js/app.js", func(w http.ResponseWriter, r *http.Request) {
		appjssource, err := ioutil.ReadFile(basepath + "js/app.js")
		if err != nil {
			panic(err)
		}
		var appjsTemplate = template.Must(template.New("").Parse(string(appjssource)))
		appjsTemplate.Execute(w, "ws://"+r.Host+"/ws")
	})
	http.HandleFunc("/js/libs/pixi.min.js", func(w http.ResponseWriter, r *http.Request) {
		pixijssource, err := ioutil.ReadFile(basepath + "js/libs/pixi.min.js")
		if err != nil {
			panic(err)
		}
		w.Write(pixijssource)
	})
	http.HandleFunc("/js/libs/jquery.slim.min.js", func(w http.ResponseWriter, r *http.Request) {
		jqueryjssource, err := ioutil.ReadFile(basepath + "js/libs/jquery.slim.min.js")
		if err != nil {
			panic(err)
		}
		w.Write(jqueryjssource)
	})
	http.HandleFunc("/images/circle.png", func(w http.ResponseWriter, r *http.Request) {
		imagesource, err := ioutil.ReadFile(basepath + "images/circle.png")
		if err != nil {
			panic(err)
		}
		w.Write(imagesource)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		homesource, err := ioutil.ReadFile(basepath + "index.html")
		if err != nil {
			panic(err)
		}
		var homeTemplate = template.Must(template.New("").Parse(string(homesource)))
		homeTemplate.Execute(w, nil)
	})

	go http.ListenAndServe(*addr, nil)

	log.Println("Viz Listening on http://" + host + ":" + strconv.Itoa(port))
}

func getcmdenv() cmdenvironment {

	// Host

	host, exists := os.LookupEnv("HOST")
	if !exists {
		host = "127.0.0.1"
	}

	// Port
	var port int
	portstr, exists := os.LookupEnv("PORT")
	if !exists {
		port = 8080
	} else {
		portbis, err := strconv.Atoi(portstr)
		if err != nil {
			portbis = 8080
		}

		port = portbis
	}

	// Number of agents
	var nbagents int
	nbagentsstr, exists := os.LookupEnv("AGENTS")
	if !exists {
		nbagents = 2
	} else {
		nbagentsbis, err := strconv.Atoi(nbagentsstr)
		if err != nil {
			nbagentsbis = 2
		}
		nbagents = nbagentsbis
	}

	// Ticks per second
	var tps int
	tpsstr, exists := os.LookupEnv("TPS")
	if !exists {
		tps = 10
	} else {
		tpsbis, err := strconv.Atoi(tpsstr)
		if err != nil {
			tpsbis = 10
		}
		tps = tpsbis
	}

	// Agent implementation
	agentimp, exists := os.LookupEnv("AGENTIMP")
	if !exists {
		agentimp = "seeker"
	}

	return cmdenvironment{
		host:     host,
		port:     port,
		agents:   nbagents,
		agentimp: agentimp,
		tps:      tps,
	}
}

func main() {

	rand.Seed(time.Now().UnixNano())

	cmdenv := getcmdenv()

	ctx := context.Background()

	exfolder, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	stopticking := make(chan bool)

	swarm := server.NewSwarm(
		ctx,
		cmdenv.host,
		cmdenv.port,
		exfolder+"/../../agents/"+cmdenv.agentimp,
		cmdenv.agents,
		cmdenv.tps,
		stopticking,
	)

	for i := 0; i < cmdenv.agents; i++ {
		go swarm.Spawnagent()
	}

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-hassigtermed
		stopticking <- true
		swarm.Teardown()
		os.Exit(1)
	}()

	go visualization(swarm, cmdenv.host, cmdenv.port+1)

	swarm.Listen()

}
