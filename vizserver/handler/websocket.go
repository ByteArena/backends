package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/common/recording"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/bytearena/bytearena/vizserver/types"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type wsincomingmessage struct {
	messageType int
	p           []byte
	err         error
}

// Simplified version of the VizMessage struct
type ArenaIdVizMessage struct {
	ArenaId string
}

func Websocket(arenas *types.VizArenaMap, recorder recording.Recorder) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		arena := arenas.Get(vars["id"])

		defer recorder.Close()

		if arena == nil {
			w.Write([]byte("ARENA NOT FOUND !"))
			return
		}

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}

		watcher := types.NewWatcher(c)
		arena.SetWatcher(watcher)

		defer func(c *websocket.Conn) {
			arena.RemoveWatcher(watcher.GetId())
			log.Println(arena.GetNumberWatchers())
			c.Close()
			log.Println("Closing !!!")
		}(c)

		/////////////////////////////////////////////////////////////
		/////////////////////////////////////////////////////////////
		/////////////////////////////////////////////////////////////

		clientclosedsocket := make(chan bool)
		c.SetCloseHandler(func(code int, text string) error {
			clientclosedsocket <- true
			return nil
		})

		// Listen to messages incoming from viz; mandatory to notice when websocket is closed client side
		incomingmsg := make(chan wsincomingmessage)
		go func(client *websocket.Conn, ch chan wsincomingmessage) {
			messageType, p, err := client.ReadMessage()
			ch <- wsincomingmessage{messageType, p, err}
		}(c, incomingmsg)

		// Listen to viz messages coming from arenaserver
		vizmsgchan := make(chan interface{})
		notify.Start("viz:message", vizmsgchan)

		for {
			select {
			case <-clientclosedsocket:
				{
					log.Println("<-clientclosedsocket")
					return
				}
			case vizmsg := <-vizmsgchan:
				{
					vizmsgString, ok := vizmsg.(string)
					utils.Assert(ok, "Failed to cast vizmessage into string")

					var vizMessage []ArenaIdVizMessage
					err := json.Unmarshal([]byte(vizmsgString), &vizMessage)

					if err != nil {
						log.Println("Failed to decode vizmessage:", err)
					} else {

						recorder.Record(vizMessage[0].ArenaId, vizmsgString)

						if arena.GetId() == vizMessage[0].ArenaId {

							// TODO: better management of message type encapsulation
							c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("{\"type\":\"framebatch\", \"data\": %s}", vizmsgString)))
						}
					}
				}
			}
		}

		/////////////////////////////////////////////////////////////
		/////////////////////////////////////////////////////////////
		/////////////////////////////////////////////////////////////
	}
}
