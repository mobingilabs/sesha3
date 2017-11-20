package session

import (
	"bufio"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/awsports"
	"github.com/mobingilabs/sesha3/pkg/metrics"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/mobingilabs/sesha3/pkg/util"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

type Session struct {
	id        string
	Online    bool
	TtyURL    string
	Cmd       *exec.Cmd
	PemFile   string
	HttpsPort string
	User      string
	Ip        string
	StackId   string
	Timeout   string
	portReq   *awsports.SecurityGroupRequest
}

func (s *Session) Id() string {
	if s.id == "" {
		u1 := uuid.NewV4()
		s.id = fmt.Sprintf("%s", u1)
	}

	return s.id
}

// Start initializes an instance of gotty and return the url.
func (s *Session) Start() (string, error) {
	// set member 'id'
	s.info("starting session: ", s.Id())

	// try to open port for gotty
	ec2req := awsports.Make(params.CredProfile, params.Region, params.Ec2Id)
	s.portReq = &ec2req
	s.HttpsPort = fmt.Sprint(ec2req.RequestPort)
	ttyurl := make(chan string)
	wsclose := make(chan string)

	// port closer function
	fnClosePort := func() {
		err := ec2req.ClosePort()
		if err != nil {
			notify.HookPost(err)
			s.error(errors.Wrap(err, "close port failed"))
		}
	}

	go func() {
		metrics.MetricsCurrentConnection.Add(1)
		defer metrics.MetricsCurrentConnection.Add(-1)

		err := ec2req.OpenPort()
		if err != nil {
			notify.HookPost(err)
			s.error(errors.Wrap(err, "open port failed"))
		}

		svrtool := cmdline.Dir() + "/tools/" + runtime.GOOS + "/gotty"
		certpath := "/etc/letsencrypt/live/" + util.Domain()
		ssh := "/usr/bin/ssh -oStrictHostKeyChecking=no -i " + s.PemFile + " " + s.User + "@" + s.Ip
		shell := "grep " + s.User + " /etc/passwd | cut -d: -f7"
		dshellb, err := exec.Command("bash", "-c", ssh+" -t "+shell).Output()
		if err != nil {
			notify.HookPost(err)
			s.error(errors.Wrap(err, "ssh shell exec failed"))
		}

		s.info(ssh + " -t " + shell)
		s.info("shell-out: ", string(dshellb))
		defaultshell := strings.TrimSpace(string(dshellb))
		s.info("default: ", defaultshell)
		timeout := "export TMOUT=" + s.Timeout
		term := "export TERM=xterm"
		s.Cmd = exec.Command(svrtool,
			"--port", fmt.Sprint(ec2req.RequestPort),
			"-w",
			"--random-url",
			"--random-url-length", "36",
			"--timeout", "30",
			"-once",
			"--tls",
			"--tls-crt",
			certpath+"/fullchain.pem",
			"--tls-key",
			certpath+"/privkey.pem",
			"--title-format", "Mobingi - {{ .Command }}",
			"bash",
			"-c",
			ssh+" -t \""+timeout+" && "+term+" && "+defaultshell+" --login \"",
		)

		s.info("svrtool args: ", s.Cmd.Args)
		errpipe, err := s.Cmd.StderrPipe()
		if err != nil {
			notify.HookPost(err)
			s.error(errors.Wrap(err, "stderr pipe connect failed"))
			fnClosePort()
		}

		s.Cmd.Start()

		go func() {
			s.info("start pipe to stderr")
			errscan := bufio.NewScanner(errpipe)
			found := false
			for {
				chk := errscan.Scan()
				if !chk {
					if errscan.Err() != nil {
						err := errors.Wrap(err, "stderr scan failed")
						notify.HookPost(err)
						s.error(err)
					}

					s.info("end stderr pipe")
					break
				}

				stxt := errscan.Text()
				s.info("scan[errpipe]: ", stxt)

				if !found {
					if strings.Index(stxt, "URL") != -1 {
						tmpurl := stxt
						ttyurl <- strings.Split(tmpurl, "URL: ")[1]
						s.info("url found")
						found = true
					}
				}

				if strings.Index(stxt, "Connection closed") != -1 {
					wsclose <- stxt
				}
			}
		}()

		err = s.Cmd.Wait()
		if err != nil {
			notify.HookPost(err)
			s.error(errors.Wrap(err, "cmd wait failed"))
		}

		fnClosePort()
		Sessions.Remove(s.Id())
		wsclose <- "__closed__"
		s.info("gotty done")
	}()

	ret := <-ttyurl

	// workaround for websocket close not exiting gotty immediately
	go func() {
		for wsc := range wsclose {
			switch wsc {
			case "__closed__":
				s.info("gotty closed normally")
			default:
				s.info("close detected: [", wsc, "]")
				time.Sleep(time.Second * 2)

				// close aws port before terminate
				s.info("attempt close port: ", s.HttpsPort)
				if s.portReq != nil {
					err := s.portReq.ClosePort()
					if err != nil {
						notify.HookPost(err)
						s.error(errors.Wrap(err, "close port failed"))
					}
				}

				// attempt to kill gotty process
				s.info("attempt to close gotty with pid: ", s.Cmd.Process.Pid)
				err := s.Cmd.Process.Signal(syscall.SIGTERM)
				if err != nil {
					s.error(errors.Wrap(err, "sigterm failed"))
					// when all else fail
					err = s.Cmd.Process.Signal(syscall.SIGKILL)
					if err != nil {
						s.error(errors.Wrap(err, "sigkill failed"))
					}
				}
			}
		}
	}()

	return ret, nil
}

func (s *Session) GetFullURL() string {
	var furl string
	if !s.Online {
		return furl
	}

	rurl, err := url.Parse(s.TtyURL)
	if err != nil {
		notify.HookPost(err)
		return furl
	}

	furl += "https://" + util.Domain() + ":" + s.HttpsPort + rurl.EscapedPath()
	return furl
}

func (s *Session) info(v ...interface{}) {
	m := fmt.Sprint(v...)
	d.Info("["+s.Id()+"]", m)
}

func (s *Session) error(v ...interface{}) {
	m := fmt.Sprint(v...)
	d.Error("["+s.Id()+"]", m)
}
