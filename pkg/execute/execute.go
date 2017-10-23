package execute

import (
	"bytes"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"os"
	"os/exec"
	"regexp"
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
	rep := regexp.MustCompile(`^\n|^\r|\n$|\r$`)
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
				out.Stderr = rep.ReplaceAllString(scpe, "")
				ret = append(ret, out)
				wg.Done()
			} else {
				execScript := exec.Command(
					"/usr/bin/ssh",
					"-o",
					"StrictHostKeyChecking=no",
					"-i", pemfile,
					data["user"].(string)+"@"+ip,
					"/tmp/"+data["script_name"].(string),
				)
				scriptout, scripterr, err := execmd(execScript)
				if err != nil {
					d.Error("script:", err)
				}
				out.Stdout = rep.ReplaceAllString(strings.Replace(scriptout, "\r", "\n", -1), "")
				ste := strings.Split(strings.Replace(scripterr, "\r", "\n", -1), "\n")
				out.Stderr = rep.ReplaceAllString(strings.Join(ste[0:len(ste)-1], "\n"), "")
				d.Info(out.Stdout)
				ret = append(ret, out)
				wg.Done()
			}
		}()
	}
	wg.Wait()
	os.Remove(data["scriptfilepath"].(string))
	return ret
}
