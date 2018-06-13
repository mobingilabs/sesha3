package session

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
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
	sshuser := s.User

	// try to open port for gotty
	ec2req := awsports.Make(util.GetRegion(), util.GetEc2Id())
	s.portReq = &ec2req
	s.HttpsPort = fmt.Sprint(ec2req.RequestPort)
	ttyurl := make(chan string)
	wsclose := make(chan string)

	if sshuser == "" || sshuser == "<nil>" {
		// AWS stack id.
		if strings.HasPrefix(s.StackId, "mo-") {
			sshuser = "ec2-user"
		}
	}

	glog.V(1).Infof("request port: %v", s.HttpsPort)

	// port closer function
	fnClosePort := func() {
		err := ec2req.ClosePort()
		if err != nil {
			s.error(util.ErrV(err, "close port failed"))
			notify.HookPost(err)
		}
	}

	go func() {
		metrics.MetricsCurrentConnection.Add(1)
		defer metrics.MetricsCurrentConnection.Add(-1)

		// make sure to open port first
		err := ec2req.OpenPort()
		if err != nil {
			s.error(util.ErrV(err, "open port failed"))
			notify.HookPost(err)
		}

		svrtool := cmdline.Dir() + "/tools/" + runtime.GOOS + "/gotty"
		certpath := "/etc/letsencrypt/live/" + util.Domain()
		ssh := "/usr/bin/ssh -oStrictHostKeyChecking=no -i " + s.PemFile + " " + sshuser + "@" + s.Ip
		shell := "grep " + sshuser + " /etc/passwd | cut -d: -f7"
		dshellb, err := exec.Command("bash", "-c", ssh+" -t "+shell).Output()
		if err != nil {
			s.error(util.ErrV(err, "ssh shell exec failed"))
			notify.HookPost(err)
		}

		// Diego request: 2018/01/09:
		// change default timeout to 2hrs
		// TODO: user should be able to configure timeout in alm.
		s.info(ssh + " -t " + shell)
		s.info("shell-out: ", string(dshellb))
		defaultshell := strings.TrimSpace(string(dshellb))
		s.info("default: ", defaultshell)
		// timeout := "export TMOUT=" + s.Timeout
		timeout := "export TMOUT=7200"
		term := "export TERM=xterm"

		if params.UseProxy {
			s.Cmd = exec.Command(svrtool,
				"--port", fmt.Sprint(ec2req.RequestPort),
				"-w",
				"--timeout", "7200",
				"-once",
				"--title-format", "Mobingi - {{ .Command }}",
				"bash",
				"-c",
				ssh+" -t \""+timeout+" && "+term+" && "+defaultshell+" --login \"",
			)
		} else {
			s.Cmd = exec.Command(svrtool,
				"--port", fmt.Sprint(ec2req.RequestPort),
				"-w",
				"--random-url",
				"--random-url-length", "36",
				"--timeout", "7200",
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
		}

		s.info("svrtool args: ", s.Cmd.Args)
		errpipe, err := s.Cmd.StderrPipe()
		if err != nil {
			s.error(util.ErrV(err, "stderr pipe connect failed"))
			notify.HookPost(err)
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
			err = errors.Wrap(err, "cmd wait failed")
			notify.HookPost(err)
			s.error(err)
		}

		fnClosePort()
		Sessions.Remove(s.Id())
		wsclose <- "__closed__"
		s.info("gotty done")
	}()

	ret := <-ttyurl

	glog.V(2).Infof("extracted url from pipe: %v", ret)

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
						err = errors.Wrap(err, "close port failed")
						notify.HookPost(err)
						s.error(err)
					}
				}

				// attempt to kill gotty process
				s.info("attempt to close gotty with pid: ", s.Cmd.Process.Pid)
				err := s.Cmd.Process.Signal(syscall.SIGTERM)
				if err != nil {
					s.error(errors.Wrap(err, "sigterm failed"))
					// when all else fail
					_ = s.Cmd.Process.Signal(syscall.SIGKILL)
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

	proxy := "https://mochi-sesha3mapperdev.mobingi.com"
	if !params.IsDev {
		proxy = "https://mochi-sesha3mapper.mobingi.com"
	}

	if params.UseProxy {
		iid := strings.Replace(util.GetEc2Id(), "-", "", -1)
		md5rand := fmt.Sprintf("%x", md5.Sum([]byte(iid+s.HttpsPort)))
		furl = proxy + "/sesha3-" + iid + "/" + s.HttpsPort + "/" + md5rand + "/"
		glog.V(2).Infof("full url: %v", furl)
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
	glog.V(1).Infof("[session:%v] %v", s.Id(), m)
}

func (s *Session) error(v ...interface{}) {
	m := fmt.Sprint(v...)
	glog.Errorf("[session:%v] %v", s.Id(), m)
}
