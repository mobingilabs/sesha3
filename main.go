package main

import (
	"github.com/mobingilabs/sesha3/cmd"
	"github.com/mobingilabs/sesha3/pkg/signal"
)

func main() {
	signal.SignalHandler()
	cmd.Execute()
}
