package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/awsports"
	"github.com/pkg/errors"
)

type session struct {
	Online    bool
	TtyURL    string
	Cmd       *exec.Cmd
	HttpsPort string
	User      string
	Ip        string
	StackId   string
	Timeout   string
}

// Start initializes an instance of gotty and return the url.
func (c *session) Start() (string, error) {
	ec2req := awsports.Make(credprof, region, ec2id)
	c.HttpsPort = fmt.Sprint(ec2req.RequestPort)
	ttyurl := make(chan string)
	wsclose := make(chan string)

	go func() {
		err := ec2req.OpenPort()
		if err != nil {
			d.Error(errors.Wrap(err, "open port failed"))
		}

		svrtool := cmdline.Dir() + "/tools/" + runtime.GOOS + "/gotty"
		certpath := cmdline.Dir() + "/certs/"
		ssh := "/usr/bin/ssh -oStrictHostKeyChecking=no -i " + os.TempDir() + "/user/" + c.StackId + ".pem " + c.User + "@" + c.Ip
		shell := "grep " + c.User + " /etc/passwd | cut -d: -f7"
		dshellb, err := exec.Command("bash", "-c", ssh+" -t "+shell).Output()
		if err != nil {
			d.Error(errors.Wrap(err, "ssh shell exec failed"))
		}

		d.Info(ssh + " -t " + shell)
		d.Info("shell-out:", string(dshellb))
		defaultshell := strings.TrimSpace(string(dshellb))
		d.Info("default:", defaultshell)
		timeout := "export TMOUT=" + c.Timeout
		term := "export TERM=xterm"
		c.Cmd = exec.Command(svrtool,
			"--port", fmt.Sprint(ec2req.RequestPort),
			"-w",
			"--random-url",
			"--random-url-length", "36",
			"-once",
			"--tls",
			"--tls-crt",
			certpath+"fullchain.pem",
			"--tls-key",
			certpath+"privkey.pem",
			"--title-format", "Mobingi - {{ .Command }}",
			"bash",
			"-c",
			ssh+" -t \""+timeout+" && "+term+" && "+defaultshell+" --login \"",
		)

		errpipe, err := c.Cmd.StderrPipe()
		if err != nil {
			d.Error(errors.Wrap(err, "stderr pipe connect failed"))
			err = ec2req.ClosePort()
			if err != nil {
				d.Error(errors.Wrap(err, "close port failed"))
			}
		}

		c.Cmd.Start()

		go func() {
			d.Info("start pipe to stderr")
			errscan := bufio.NewScanner(errpipe)
			found := false
			for {
				chk := errscan.Scan()
				if !chk {
					if errscan.Err() != nil {
						d.Error(errors.Wrap(err, "stderr scan failed"))
					}

					d.Info("end stderr pipe")
					break
				}

				stxt := errscan.Text()
				d.Info("scan[errpipe]:", stxt)

				if !found {
					if strings.Index(stxt, "URL") != -1 {
						tmpurl := stxt
						ttyurl <- strings.Split(tmpurl, "URL: ")[1]
						d.Info("url found")
						found = true
					}
				}

				if strings.Index(stxt, "websocket: close") != -1 {
					wsclose <- stxt
				}
			}
		}()

		c.Cmd.Wait()
		err = ec2req.ClosePort()
		if err != nil {
			d.Error(errors.Wrap(err, "close port failed"))
		}

		wsclose <- "__closed__"
		d.Info("gotty done")

		// delete pem file when done
		err = os.Remove(os.TempDir() + "/user/" + c.StackId + ".pem")
		if err != nil {
			d.Error(errors.Wrap(err, "delete pem failed"))
		}
	}()

	ret := <-ttyurl

	// workaround for websocket close not exiting gotty immediately
	go func() {
		wsc := <-wsclose
		if wsc != "__closed__" {
			d.Info("websocket close detected:", wsc)
			d.Info("attempt to close gotty...")
			time.Sleep(time.Second * 1)
			err := c.Cmd.Process.Signal(syscall.SIGTERM)
			if err != nil {
				d.Error(errors.Wrap(err, "sigterm failed"))
				err = c.Cmd.Process.Signal(syscall.SIGKILL)
				if err != nil {
					d.Error(errors.Wrap(err, "sigkill failed"))
				}
			}
		} else {
			d.Info("gotty closed normally")
		}
	}()

	return ret, nil
}

func (c *session) GetFullURL() string {
	var furl string
	if !c.Online {
		return furl
	}

	rurl, err := url.Parse(c.TtyURL)
	if err != nil {
		return furl
	}

	furl += "https://" + domain + ":" + c.HttpsPort + rurl.EscapedPath()
	return furl
}
