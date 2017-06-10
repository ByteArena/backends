package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	log.Println("Byte Arena Trainer v0.1")

	tickspersec := flag.Int("tps", 10, "Number of ticks per second")
	host := flag.String("host", "", "IP serving the trainer; required")
	port := flag.Int("port", 8080, "Port serving the trainer")

	flag.Parse()

	if *host == "" {
		fmt.Println("-host is required")
		os.Exit(1)
	}

	trainer := NewTrainingServer(*host, *port, *tickspersec)
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
