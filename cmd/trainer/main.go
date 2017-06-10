package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	log.Println("Byte Arena Trainer v0.1")
	tickspersec := 10

	trainer := NewTrainingServer(tickspersec)
	trainer.RegisterAgent("registry.bytearena.com/xtuc/test")

	// handling signals
	hassigtermed := make(chan os.Signal, 2)
	signal.Notify(hassigtermed, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-hassigtermed
		trainer.Stop()
	}()

	<-trainer.Start()
	trainer.TearDown()
}
