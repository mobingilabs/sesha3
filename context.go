package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mobingilabs/sesha3/awsports"
	"github.com/pkg/errors"
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
	name, err := os.Executable()
	if err != nil {
		return ret, errors.Wrap(err, "get executable name failed")
	}

	log.Println("name:", name)

	c.HttpsPort = fmt.Sprint(ec2req.RequestPort)

	ttyurl := make(chan string)
	go func() {
		ec2req.Openport()
		svrtool := filepath.Dir(name) + "/tools/" + runtime.GOOS + "/gotty"
		certpath := filepath.Dir(name) + "/certs/"
		ssh := "/usr/bin/ssh -oStrictHostKeyChecking=no -i " + "./tmp/" + c.StackId + ".pem " + c.User + "@" + c.Ip
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
			log.Println(err)
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
			log.Println(scanner.Text())
		}
		ttyurl <- out
		c.Cmd.Wait()
		ec2req.Closeport()
		log.Println("gotty finish!")
		err = os.Remove("./tmp/" + c.StackId + ".pem")
		log.Println("Delete!", err)
		if err != nil {
			log.Println(err)
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
