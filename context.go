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
)

/*
type message struct {
	Pem     string
	User    string
	Ip      string
	Stackid string
	Timeout string
	Err     int
}
*/

type context struct {
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
func (c *context) Start() (ret string, err error) {
	ec2req := awsports.Make(credprof, region, ec2id)
	c.HttpsPort = fmt.Sprint(ec2req.RequestPort)
	ttyurl := make(chan string)
	go func() {
		err := ec2req.Openport()
		if err != nil {
			d.Error(err)
		}

		svrtool := cmdline.Dir() + "/tools/" + runtime.GOOS + "/gotty"
		certpath := cmdline.Dir() + "/certs/"
		ssh := "/usr/bin/ssh -oStrictHostKeyChecking=no -i " + os.TempDir() + "/" + c.StackId + ".pem " + c.User + "@" + c.Ip
		shell := "grep ec2-user /etc/passwd | cut -d: -f7"
		dshellb, _ := exec.Command("bash", "-c", ssh+" -t "+shell).Output()
		defaultshell := strings.TrimSpace(string(dshellb))
		timeout := c.Timeout
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
			ssh+" -t \"export TMOUT="+timeout+" && "+defaultshell+" --login \"",
		)

		errpipe, err := c.Cmd.StderrPipe()
		if err != nil {
			d.Error(err)
			ec2req.Closeport()
		}

		c.Cmd.Start()
		scanner := bufio.NewScanner(errpipe)
		out := ""
		for scanner.Scan() {
			if strings.Index(scanner.Text(), "URL") != -1 {
				tmpurl := scanner.Text()
				out = strings.Split(tmpurl, "URL: ")[1]
				break
			}

			d.Info(scanner.Text())
		}

		ttyurl <- out
		c.Cmd.Wait()
		ec2req.Closeport()
		d.Info("gotty done")

		// delete pem file when done
		err = os.Remove(os.TempDir() + "/" + c.StackId + ".pem")
		if err != nil {
			d.Error(err)
		} else {
			d.Info("gotty done")
		}
	}()

	ret = <-ttyurl
	return
}

func (c *context) GetFullURL() string {
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
