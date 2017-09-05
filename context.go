package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"

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
	Mu        sync.Mutex
	Online    bool
	ClientURL string
	TtyURL    string
	Cmd       *exec.Cmd
	HttpsPort string
}

// Start initializes an instance of gotty and return the url.
func (c *context) Start(get message) error {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	c.kill()

	ec2req := awsports.Make(devprofile, awsRegion, devinst)
	name, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "get executable name failed")
	}

	log.Println("name:", name)

	c.HttpsPort = fmt.Sprint(ec2req.RequestPort)

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
			"--notify-url", "http://"+devdomain+":"+httpPort+"/hook",
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

		con, err := c.Cmd.CombinedOutput()
		if err != nil {
			fmt.Println(err)
		}
		ec2req.Closeport()
		fmt.Println(string(con), "finish!")
		err = os.Remove("./tmp/" + get.Stackid + ".pem")
		fmt.Println("Delete!", err)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(string(con))
	}()

	return nil
}

func (c *context) SetClientURL(cu string) {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	c.ClientURL = cu
}

func (c *context) kill() {
	c.Online = false
	c.ClientURL = ""
	if c.Cmd == nil {
		return
	}

	if c.Cmd.Process == nil {
		return
	}

	if c.Cmd.Process.Pid > 0 {
		err := c.Cmd.Process.Signal(syscall.Signal(syscall.SIGTERM))
		log.Printf("SIGTERM on pid %d returned: %v\n", c.Cmd.Process.Pid, err)
		if err != nil {
			err = c.Cmd.Process.Signal(syscall.Signal(syscall.SIGKILL))
			log.Printf("SIGKILL on pid %d returned: %v\n", c.Cmd.Process.Pid, err)
		}
	}
}

func (c *context) SetRandomURL(ru string) {
	c.Mu.Lock()
	defer c.Mu.Unlock()
	c.TtyURL = ru
	c.Online = true
}

func (c *context) GetFullURL() string {
	c.Mu.Lock()
	defer c.Mu.Unlock()

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
