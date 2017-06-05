package main

import (
	"log"
	"net/http"

	notify "github.com/bitly/go-notify"
	"github.com/bytearena/bytearena/cmd/viz-server/types"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type wsincomingmessage struct {
	messageType int
	p           []byte
	err         error
}

func websocketHandler(arenas *types.VizArenaMap) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		arena := arenas.Get(vars["id"])

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

		incomingmsg := make(chan wsincomingmessage)
		go func(client *websocket.Conn, ch chan wsincomingmessage) {
			messageType, p, err := client.ReadMessage()
			ch <- wsincomingmessage{messageType, p, err}
		}(c, incomingmsg)

		vizmsgchan := make(chan interface{})
		notify.Start("viz:message", vizmsgchan)

		for {
			select {
			// case <-incomingmsg:
			// 	{
			// 		log.Println(incomingmsg)
			// 	}
			case <-clientclosedsocket:
				{
					log.Println("<-clientclosedsocket")
					return
				}
			case vizmsg := <-vizmsgchan:
				{
					if json, ok := vizmsg.(string); ok {
						c.WriteMessage(websocket.TextMessage, []byte(json))
					}
				}
			}
		}

		/////////////////////////////////////////////////////////////
		/////////////////////////////////////////////////////////////
		/////////////////////////////////////////////////////////////
	}
}
