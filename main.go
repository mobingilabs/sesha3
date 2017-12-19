package main

import (
	"github.com/golang/glog"
	"github.com/mobingilabs/sesha3/cmd"
	"github.com/mobingilabs/sesha3/pkg/signal"
)

func main() {
	glog.CopyStandardLogTo("INFO")
	signal.SignalHandler()
	cmd.Execute()
}
