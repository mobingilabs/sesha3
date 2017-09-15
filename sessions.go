package main

import (
	"sync"
	"syscall"

	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/pkg/errors"
)

type sessions struct {
	sync.Mutex
	ss []session
}

func (s *sessions) Add(item session) {
	s.Lock()
	defer s.Unlock()
	s.ss = append(s.ss, item)
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
	return nil
}

func (s *sessions) Count() int {
	s.Lock()
	defer s.Unlock()
	return len(s.ss)
}

func (s *sessions) TerminateAll() []error {
	s.Lock()
	defer s.Unlock()
	ret := make([]error, 0)
	for _, sess := range s.ss {
		err := sess.Cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			err := errors.Wrap(err, "sigterm failed")
			ret = append(ret, err)
			d.Error(err)
			// when all else fail
			sess.Cmd.Process.Signal(syscall.SIGKILL)
		}
	}

	return ret
}
