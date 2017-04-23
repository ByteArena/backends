package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/netgusto/bytearena/server"
	"github.com/netgusto/bytearena/server/state"
	"github.com/netgusto/bytearena/utils/vector"
	uuid "github.com/satori/go.uuid"
)

type vizmessage struct {
	Agents          []vizagentmessage
	Projectiles     []vizprojectilemessage
	Obstacles       []vizobstaclemessage
	DebugIntersects []vector.Vector2
}

type vizagentmessage struct {
	Id           uuid.UUID
	X            float64
	Y            float64
	Position     vector.Vector2
	VisionRadius float64
	VisionAngle  float64
	Radius       float64
	Kind         string
	Orientation  float64
}

type vizprojectilemessage struct {
	X        float64
	Y        float64
	Position vector.Vector2
	Radius   float64
	From     vizagentmessage
	Kind     string
}

type vizobstaclemessage struct {
	A vector.Vector2
	B vector.Vector2
}

type wsincomingmessage struct {
	messageType int
	p           []byte
	err         error
}

func wsendpoint(w http.ResponseWriter, r *http.Request, statechan chan state.ServerState) {

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
		case serverstate := <-statechan:
			{
				msg := vizmessage{}

				serverstate.Projectilesmutex.Lock()
				for _, projectile := range serverstate.Projectiles {
					msg.Projectiles = append(msg.Projectiles, vizprojectilemessage{
						Position: projectile.Velocity,
						Radius:   projectile.Radius,
						Kind:     "projectiles",
						From: vizagentmessage{
							Position: projectile.Position,
						},
					})
				}
				serverstate.Projectilesmutex.Unlock()

				serverstate.Agentsmutex.Lock()
				for id, agent := range serverstate.Agents {
					msg.Agents = append(msg.Agents, vizagentmessage{
						Id:           id,
						Kind:         "agent",
						Position:     agent.Position,
						Radius:       agent.Radius,
						Orientation:  agent.Orientation,
						VisionRadius: agent.VisionRadius,
						VisionAngle:  agent.VisionAngle,
					})
				}
				serverstate.Agentsmutex.Unlock()

				serverstate.Obstaclesmutex.Lock()
				for _, obstacle := range serverstate.Obstacles {
					msg.Obstacles = append(msg.Obstacles, vizobstaclemessage{
						A: obstacle.A,
						B: obstacle.B,
					})
				}
				serverstate.Obstaclesmutex.Unlock()

				msg.DebugIntersects = serverstate.DebugIntersects

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

func visualization(srv *server.Server, host string, port int) {

	basepath := "./client/"

	addr := flag.String("addr", host+":"+strconv.Itoa(port), "http service address")

	stateobserver := srv.SubscribeStateObservation()
	staterelays := make([]chan state.ServerState, 0)
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
		relay := make(chan state.ServerState)
		staterelays = append(staterelays, relay)
		relayindex := len(staterelays) - 1
		staterelaymutex.Unlock()

		wsendpoint(w, r, relay)

		staterelaymutex.Lock()
		copy(staterelays[relayindex:], staterelays[relayindex+1:])
		staterelays[len(staterelays)-1] = nil // or the zero value of T
		staterelays = staterelays[:len(staterelays)-1]
		staterelaymutex.Unlock()
	})

	http.HandleFunc("/js/app.js", func(w http.ResponseWriter, r *http.Request) {
		appjssource, err := ioutil.ReadFile(basepath + "js/app.js")
		if err != nil {
			panic(err)
		}
		var appjsTemplate = template.Must(template.New("").Parse(string(appjssource)))
		appjsTemplate.Execute(w, "ws://"+r.Host+"/ws")
	})
	http.HandleFunc("/js/vector2.js", func(w http.ResponseWriter, r *http.Request) {
		appjssource, err := ioutil.ReadFile(basepath + "js/vector2.js")
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
	http.HandleFunc("/images/triangle.png", func(w http.ResponseWriter, r *http.Request) {
		imagesource, err := ioutil.ReadFile(basepath + "images/triangle.png")
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
