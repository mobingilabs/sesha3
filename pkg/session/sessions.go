package session

import (
	"sync"
	"syscall"

	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/pkg/errors"
)

var Sessions sessions

type sessions struct {
	sync.Mutex
	ss []Session
}

func (s *sessions) Add(item Session) {
	s.Lock()
	defer s.Unlock()
	s.ss = append(s.ss, item)
	d.Info("session added:", item.Id())
}

func (s *sessions) Remove(id string) error {
	s.Lock()
	defer s.Unlock()
	var idx int = -1
	for i, sess := range s.ss {
		if sess.id == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return errors.New("id not found")
	}

	s.ss[idx] = s.ss[len(s.ss)-1] // replace it with the last one
	s.ss = s.ss[:len(s.ss)-1]     // remove the last one
	d.Info("session removed:", id)
	return nil
}

func (s *sessions) Count() int {
	s.Lock()
	defer s.Unlock()
	return len(s.ss)
}

type SessionsDescribe struct {
	Id        string `json:"id"`
	Url       string `json:"url"`
	HttpsPort string `json:"port"`
	Pid       int    `json:"pid"`
}

func (s *sessions) Describe() []SessionsDescribe {
	s.Lock()
	defer s.Lock()
	ret := make([]SessionsDescribe, 0)
	for _, sess := range s.ss {
		tmp := SessionsDescribe{
			Id:        sess.Id(),
			Url:       sess.TtyURL,
			HttpsPort: sess.HttpsPort,
			Pid:       sess.Cmd.Process.Pid,
		}

		ret = append(ret, tmp)
	}

	return ret
}

func (s *sessions) TerminateAll() []error {
	s.Lock()
	defer s.Unlock()
	ret := make([]error, 0)
	for _, sess := range s.ss {
		// close aws port before terminate
		d.Info("attempt close port:", sess.HttpsPort)
		err := sess.portReq.ClosePort()
		if err != nil {
			notify.HookPost(err)
			d.Error(err)
		}

		// try kill process
		d.Info("attempt kill pid:", sess.Cmd.Process.Pid)
		err = sess.Cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			err := errors.Wrap(err, "sigterm failed")
			if err != nil {
				d.Error(err)
			}

			ret = append(ret, err)
			// when all else fail
			sess.Cmd.Process.Signal(syscall.SIGKILL)
		}
	}

	return ret
}
