package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

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
func (c *session) Start() (ret string, err error) {
	ec2req := awsports.Make(credprof, region, ec2id)
	c.HttpsPort = fmt.Sprint(ec2req.RequestPort)
	ttyurl := make(chan string)
	go func() {
		err := ec2req.Openport()
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

		outpipe, err := c.Cmd.StdoutPipe()
		if err != nil {
			d.Error(errors.Wrap(err, "stdout pipe connect failed"))
			ec2req.Closeport()
		}

		errpipe, err := c.Cmd.StderrPipe()
		if err != nil {
			d.Error(errors.Wrap(err, "stderr pipe connect failed"))
			ec2req.Closeport()
		}

		c.Cmd.Start()

		go func() {
			d.Info("start pipe to stdout")
			outscan := bufio.NewScanner(outpipe)
			for {
				chk := outscan.Scan()
				if !chk {
					if outscan.Err() != nil {
						d.Error(errors.Wrap(err, "stdout scan failed"))
					}

					d.Info("end stdout pipe")
					break
				}

				d.Info("scan[outpipe]:", outscan.Text())
			}
		}()

		d.Info("start pipe to stderr")
		scanner := bufio.NewScanner(errpipe)
		out := ""
		for {
			chk := scanner.Scan()
			if !chk {
				if scanner.Err() != nil {
					d.Error(errors.Wrap(err, "stderr scan failed"))
				}

				d.Info("end stderr pipe")
				break
			}

			stxt := scanner.Text()
			d.Info("scan[errpipe]:", stxt)
			if strings.Index(stxt, "URL") != -1 {
				tmpurl := stxt
				out = strings.Split(tmpurl, "URL: ")[1]
				d.Info("end stderr pipe")
				break
			}
		}

		ttyurl <- out
		c.Cmd.Wait()
		ec2req.Closeport()
		d.Info("gotty done")

		// delete pem file when done
		err = os.Remove(os.TempDir() + "/user/" + c.StackId + ".pem")
		if err != nil {
			d.Error(err)
		}
	}()

	ret = <-ttyurl
	return
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
