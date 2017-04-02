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
	X     float64
	Y     float64
	Kind string
}

type wsincomingmessage struct {
	messageType int
	p           []byte
	err         error
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
						X:     x,
						Y:     y,
						Kind: "agent",
					})
				}

				x, y := swarmstate.Pin.Get()

				msg.Agents = append(msg.Agents, vizagentmessage{
					X:	 x,
					Y:	 y,
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

func visualization(swarm *server.Swarm) {

	basepath := "./client/"

	addr := flag.String("addr", "0.0.0.0:8080", "http service address")

	flag.Parse()
	log.SetFlags(0)

	stateChan := swarm.Subscribe()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsendpoint(w, r, stateChan)
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

	log.Println("Viz Listening !")
}

func main() {

	rand.Seed(time.Now().UnixNano())

	host := os.Getenv("HOST")

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		port = 8080
	}

	nbagents, err := strconv.Atoi(os.Getenv("AGENTS"))
	if err != nil {
		nbagents = 8
	}

	tickspersec, err := strconv.Atoi(os.Getenv("TPS"))
	if err != nil {
		tickspersec = 10
	}

	agentimp := os.Getenv("AGENTIMP")
	if err != nil {
		agentimp = "seeker"
	}

	ctx := context.Background()

	exfolder, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	stopticking := make(chan bool)

	swarm := server.NewSwarm(
		ctx,
		host,
		port,
		exfolder+"/../../agents/"+agentimp,
		nbagents,
		tickspersec,
		stopticking,
	)

	for i := 0; i < nbagents; i++ {
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

	go visualization(swarm)

	swarm.Listen()

}
