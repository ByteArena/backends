package handler

import (
	"fmt"
	"io/ioutil"
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

		go startStreaming(recordFile, UUID)

		// Init map
		resp, err := http.Get("http://bytearena.com/maps/deathmatch/desert/death-valley/map.json")
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)

		initMessage := "{\"type\":\"init\",\"data\": {\"map\":" + string(body) + "}}"
		c.WriteMessage(websocket.TextMessage, []byte(initMessage))

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
			}
		}
	}
}

func startStreaming(filename string, UUID string) {
	debug := false

	replay.Read(filename, debug, UUID, func(line string, debug bool, UUID string) {
		notify.PostTimeout("viz:message_replay:"+UUID, line, time.Millisecond)
		<-time.NewTimer(1 * time.Second).C
	})
}
