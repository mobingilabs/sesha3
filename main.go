package main

import (
	"log"

	"github.com/mobingilabs/sesha3/cmd"
	"github.com/mobingilabs/sesha3/pkg/signal"
)

func main() {
	log.SetFlags(0)
	signal.SignalHandler()
	cmd.Execute()
}
