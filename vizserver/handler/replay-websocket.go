package handler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	notify "github.com/bitly/go-notify"

	"github.com/bytearena/bytearena/common/recording"
	"github.com/bytearena/bytearena/common/replay"
	"github.com/bytearena/bytearena/common/utils"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func ReplayWebsocket(recorder recording.Recorder, basepath string) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		UUID := vars["recordId"]

		recordFile := recorder.GetDirectory() + "/record-" + UUID + ".bin"

		_, err := os.Stat(recordFile)

		if os.IsNotExist(err) {
			w.Write([]byte("Record not found"))
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

		defer func(c *websocket.Conn) {
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
		notify.Start("viz:message_replay:"+UUID, vizmsgchan)

		vizmapmsgchan := make(chan interface{})
		notify.Start("viz:map:"+UUID, vizmapmsgchan)

		go startStreaming(recordFile, UUID)

		for {
			select {
			case <-clientclosedsocket:
				{
					utils.Debug("ws", "disconnected")
					return
				}
			case vizmsg := <-vizmsgchan:
				{
					vizmsgString, ok := vizmsg.(string)
					utils.Assert(ok, "Failed to cast vizmessage into string")

					data := fmt.Sprintf("{\"type\":\"framebatch\", \"data\": %s}", vizmsgString)

					c.WriteMessage(websocket.TextMessage, []byte(data))
				}
			case vizmap := <-vizmsgchan:
				{
					vizmapString, ok := vizmap.(string)
					utils.Assert(ok, "Failed to cast vizmessage into string")

					initMessage := "{\"type\":\"init\",\"data\": {\"map\":" + vizmapString + "}}"
					c.WriteMessage(websocket.TextMessage, []byte(initMessage))
				}
			}
		}
	}
}

func startStreaming(filename string, UUID string) {
	debug := false

	replay.Read(filename, debug, UUID, onReplayMessage, onReplayMap)
}

func onReplayMessage(line string, debug bool, UUID string) {
	notify.PostTimeout("viz:message_replay:"+UUID, line, time.Millisecond)
	<-time.NewTimer(1 * time.Second).C
}

func onReplayMap(body string, debug bool, UUID string) {
	notify.PostTimeout("viz:map:"+UUID, body, time.Millisecond)
}
