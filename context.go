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

type message struct {
	Pem     string
	User    string
	Ip      string
	Stackid string
	Timeout string
	Err     int
}

type context struct {
	Online    bool
	TtyURL    string
	Cmd       *exec.Cmd
	HttpsPort string
}

// Start initializes an instance of gotty and return the url.
func (c *context) Start(get message) (ret string, err error) {
	ec2req := awsports.Make(devprofile, awsRegion, devinst)
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
		ssh := "/usr/bin/ssh -oStrictHostKeyChecking=no -i " + "./tmp/" + get.Stackid + ".pem " + get.User + "@" + get.Ip
		shell := "grep ec2-user /etc/passwd | cut -d: -f7"
		dshellb, _ := exec.Command("bash", "-c", ssh+" -t "+shell).Output()
		defaultshell := strings.TrimSpace(string(dshellb))
		timeout := get.Timeout
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
			fmt.Println(err)
		}
		c.Cmd.Start()
		scanner := bufio.NewScanner(errpipe)
		ec2req.Closeport()
		for scanner.Scan() {
			out := ""
			if strings.Index(scanner.Text(), "Alternative URL") != -1 {
				tmpurl := scanner.Text()
				out = strings.Split(tmpurl, "URL: ")[1]
				ttyurl <- out
				break
			}
		}
		c.Cmd.Wait()
		fmt.Println("gotty finish!")
		err = os.Remove("./tmp/" + get.Stackid + ".pem")
		fmt.Println("Delete!", err)
		if err != nil {
			fmt.Println(err)
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

	furl += "https://" + devdomain + ":" + c.HttpsPort + rurl.EscapedPath()
	return furl
}
