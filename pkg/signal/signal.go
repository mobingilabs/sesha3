package signal

import (
	"os"
	"os/signal"
	"syscall"

	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
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
				d.Info("remaining sessions:", session.Sessions.Count())
				notify.HookPost("sesha3 server is stopped.")
				errs := session.Sessions.TerminateAll()
				if len(errs) > 0 {
					d.Error(errs)
				}

				os.Exit(0)
			}
		}
	}()
}
