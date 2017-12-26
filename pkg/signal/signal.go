package signal

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/mobingilabs/sesha3/pkg/session"
)

func SignalHandler() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(
		sigchan,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	go func() {
		for {
			s := <-sigchan
			switch s {
			case syscall.SIGINT, syscall.SIGTERM:
				// try cleanup remaining sessions, if any
				glog.Infof("remaining sessions: %v", session.Sessions.Count())
				notify.HookPost("sesha3 server is stopped.")
				errs := session.Sessions.TerminateAll()
				if len(errs) > 0 {
					glog.Errorf("term all failed: %v", errs)
				}

				os.Exit(0)
			}
		}
	}()
}
