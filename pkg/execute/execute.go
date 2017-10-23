package execute

import (
	"bytes"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"os/exec"
	"strings"
	"sync"
)

func execmd(cmd *exec.Cmd) (string, string, error) {
	var outb, errb bytes.Buffer
	var stdout, stderr string
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		d.Error(err)
	}
	stdout = outb.String()
	stderr = errb.String()
	return stdout, stderr, err
}

type result struct {
	Stdout string
	Stderr string
	Ip     string
}

func Sshcmd(data map[string]interface{}) []result {
	Ips := strings.Split(data["target"].(string), ",")
	pemfile := data["pem"].(string)
	ret := []result{}

	var wg sync.WaitGroup
	for _, ip := range Ips {
		wg.Add(1)
		go func() {
			//scp sesha3 to user instance
			var out result
			out.Ip = ip
			scp := exec.Command(
				"/usr/bin/scp",
				"-p",
				"-i", pemfile,
				data["scriptfilepath"].(string),
				data["user"].(string)+"@"+ip+":/tmp/",
			)
			_, scpe, err := execmd(scp)
			if err != nil {
				out.Stderr = scpe + "\n"
			}
			//

			execScript := exec.Command(
				"/usr/bin/ssh",
				"-tt",
				"-o",
				"StrictHostKeyChecking=no",
				"-i", pemfile,
				data["user"].(string)+"@"+ip,
				"/tmp/"+data["script_name"].(string),
			)
			scriptout, scripterr, err := execmd(execScript)
			out.Stdout = out.Stdout + scriptout + "\n"
			out.Stderr = out.Stderr + scripterr + "\n"
			ret = append(ret, out)
			wg.Done()
		}()
	}
	wg.Wait()
	return ret
}
