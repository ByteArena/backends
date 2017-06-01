package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	commonprotocol "github.com/bytearena/bytearena/common/protocol"
	"github.com/bytearena/bytearena/server"
	"github.com/bytearena/bytearena/server/state"
	"github.com/bytearena/leakybucket/bucket"
	"github.com/gorilla/websocket"
)

type wsincomingmessage struct {
	messageType int
	p           []byte
	err         error
}

func wsendpoint(w http.ResponseWriter, r *http.Request, statechan chan state.ServerState, tps int) {

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { // allow Origin header (CORS)
			return true
		},
	}

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

	buk := bucket.NewBucket(tps, 10, func(batch bucket.Batch, bucket *bucket.Bucket) {

		log.Println("batch !")
		frames := batch.GetFrames()
		jsonbatch := make([]json.RawMessage, len(frames))
		for i, frame := range frames {
			jsonbatch[i] = json.RawMessage(frame.GetPayload())
		}

		jsonstring, err := json.Marshal(jsonbatch)

		err = c.WriteMessage(websocket.TextMessage, jsonstring)
		if err != nil {
			log.Println("write error !")
			return
		}
	})

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
				msg := commonprotocol.VizMessage{}

				serverstate.Projectilesmutex.Lock()
				for _, projectile := range serverstate.Projectiles {
					msg.Projectiles = append(msg.Projectiles, commonprotocol.VizProjectileMessage{
						Position: projectile.Velocity,
						Radius:   projectile.Radius,
						Kind:     "projectiles",
						From: commonprotocol.VizAgentMessage{
							Position: projectile.Position,
						},
					})
				}
				serverstate.Projectilesmutex.Unlock()

				serverstate.Agentsmutex.Lock()
				for id, agent := range serverstate.Agents {
					msg.Agents = append(msg.Agents, commonprotocol.VizAgentMessage{
						Id:           id,
						Kind:         "agent",
						Position:     agent.Position,
						Velocity:     agent.Velocity,
						Radius:       agent.Radius,
						Orientation:  agent.Orientation,
						VisionRadius: agent.VisionRadius,
						VisionAngle:  agent.VisionAngle,
					})
				}
				serverstate.Agentsmutex.Unlock()

				serverstate.Obstaclesmutex.Lock()
				for _, obstacle := range serverstate.Obstacles {
					msg.Obstacles = append(msg.Obstacles, commonprotocol.VizObstacleMessage{
						A: obstacle.A,
						B: obstacle.B,
					})
				}
				serverstate.Obstaclesmutex.Unlock()

				msg.DebugIntersects = serverstate.DebugIntersects
				msg.DebugIntersectsRejected = serverstate.DebugIntersectsRejected
				msg.DebugPoints = serverstate.DebugPoints

				json, err := json.Marshal(msg)
				if err != nil {
					log.Println("json error, wtf")
					return
				}

				buk.AddFrame(string(json))
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

		wsendpoint(w, r, relay, srv.GetTicksPerSecond())

		staterelaymutex.Lock()
		copy(staterelays[relayindex:], staterelays[relayindex+1:])
		staterelays[len(staterelays)-1] = nil // or the zero value of T
		staterelays = staterelays[:len(staterelays)-1]
		staterelaymutex.Unlock()
	})

	staticfile := func(relfile string) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.Host)
			if strings.Contains(relfile, ".js") {
				w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			}

			appjssource, err := ioutil.ReadFile(basepath + relfile)
			if err != nil {
				panic(err)
			}

			if relfile == "index.html" {
				//arenaspecs := server.
				arenaspecs := srv.GetArena().GetSpecs()
				var appjsTemplate = template.Must(template.New("").Parse(string(appjssource)))
				appjsTemplate.Execute(w, struct {
					Host        string
					ArenaWidth  int
					ArenaHeight int
					ArenaName   string
					Tps         int
				}{
					r.Host,
					arenaspecs.Surface.Width.RoundPixels(),
					arenaspecs.Surface.Height.RoundPixels(),
					arenaspecs.Name,
					srv.GetTicksPerSecond(),
				})
			} else {
				w.Write(appjssource)
			}
		}
	}

	http.HandleFunc("/js/comm.js", staticfile("js/comm.js"))
	http.HandleFunc("/js/app.js", staticfile("js/app.js"))
	http.HandleFunc("/node_modules/bytearena-sdk/lib/browser/bytearenasdk.min.js", staticfile("node_modules/bytearena-sdk/lib/browser/bytearenasdk.min.js"))
	http.HandleFunc("/js/libs/pixi.min.js", staticfile("js/libs/pixi.min.js"))
	http.HandleFunc("/js/libs/jquery.slim.min.js", staticfile("js/libs/jquery.slim.min.js"))
	http.HandleFunc("/images/triangle.png", staticfile("images/triangle.png"))
	http.HandleFunc("/", staticfile("index.html"))

	go http.ListenAndServe(*addr, nil)

	log.Println("Viz Listening on http://" + host + ":" + strconv.Itoa(port))
}
