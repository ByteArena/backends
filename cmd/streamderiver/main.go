package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type wsincomingmessage struct {
	messageType int
	p           []byte
	err         error
}

var addr = flag.String("addr", "0.0.0.0:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

var tickspersec = 60
var tickduration = time.Duration((1000 / time.Duration(tickspersec)) * time.Millisecond)
var ticker = time.Tick(tickduration)

func wsendpoint(w http.ResponseWriter, r *http.Request) {

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

	centerx := 300.0
	centery := 300.0
	radius := 120.0
	frame := 0.0

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
		case <-ticker:
			{
				x := centerx + radius*math.Cos(frame)
				y := centery + radius*math.Sin(frame)
				//log.Println("Tick")
				err := c.WriteMessage(websocket.TextMessage, []byte("["+fmt.Sprintf("%.4f", x)+", "+fmt.Sprintf("%.4f", y)+"]"))
				if err != nil {
					log.Println("write error !")
					return
				}

				frame += 0.05
			}
		}
	}

}

func main() {
	flag.Parse()
	log.SetFlags(0)

	http.HandleFunc("/ws", wsendpoint)
	http.HandleFunc("/js/app.js", func(w http.ResponseWriter, r *http.Request) {
		appjssource, err := ioutil.ReadFile("client/js/app.js")
		if err != nil {
			panic(err)
		}
		var appjsTemplate = template.Must(template.New("").Parse(string(appjssource)))
		appjsTemplate.Execute(w, "ws://"+r.Host+"/ws")
	})
	http.HandleFunc("/js/libs/pixi.min.js", func(w http.ResponseWriter, r *http.Request) {
		pixijssource, err := ioutil.ReadFile("client/js/libs/pixi.min.js")
		if err != nil {
			panic(err)
		}
		w.Write(pixijssource)
	})
	http.HandleFunc("/js/libs/jquery.slim.min.js", func(w http.ResponseWriter, r *http.Request) {
		jqueryjssource, err := ioutil.ReadFile("client/js/libs/jquery.slim.min.js")
		if err != nil {
			panic(err)
		}
		w.Write(jqueryjssource)
	})
	http.HandleFunc("/images/circle.png", func(w http.ResponseWriter, r *http.Request) {
		imagesource, err := ioutil.ReadFile("client/images/circle.png")
		if err != nil {
			panic(err)
		}
		w.Write(imagesource)
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		homesource, err := ioutil.ReadFile("client/index.html")
		if err != nil {
			panic(err)
		}
		var homeTemplate = template.Must(template.New("").Parse(string(homesource)))
		homeTemplate.Execute(w, nil)
	})

	go http.ListenAndServe(*addr, nil)

	log.Println("Listening !")

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)
	<-hassigtermed
}
