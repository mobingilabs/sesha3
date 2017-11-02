package execute

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
)

func execmd(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
	/*
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
	*/
}

type Result struct {
	Out     string
	Ip      string
	Stackid string
}

func Sshcmd(stackid string, data map[string]interface{}) []Result {
	ips := data["target"].([]string)
	pemfile := data["pem"].(string)
	ret := []Result{}
	d.Info("exec:", ips)

	var wg sync.WaitGroup
	rep := regexp.MustCompile(`^\n|^\r|\n$|\r$`)
	for _, ip := range ips {
		wg.Add(1)
		go func() {
			// scp sesha3 to user instance
			var out Result
			out.Ip = ip
			out.Stackid = stackid
			scp := exec.Command(
				"/usr/bin/scp",
				"-p",
				"-i", pemfile,
				"-o", "StrictHostKeyChecking=no",
				data["scriptfilepath"].(string),
				data["user"].(string)+"@"+ip+":/tmp/",
			)

			d.Info("run-scp:", scp.Args)
			scpb, err := execmd(scp)
			if err != nil {
				out.Out = rep.ReplaceAllString(string(scpb), "")
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

				d.Info("run-ssh:", execScript.Args)
				scriptout, err := execmd(execScript)
				if err != nil {
					d.Error("script:", err)
				}

				out.Out = rep.ReplaceAllString(strings.Replace(string(scriptout), "\r", "\n", -1), "")
				// ste := strings.Split(strings.Replace(scripterr, "\r", "\n", -1), "\n")
				// out.Stderr = rep.ReplaceAllString(strings.Join(ste[0:len(ste)-1], "\n"), "")
				d.Info("out:", out.Out)
				ret = append(ret, out)
				wg.Done()
			}
		}()
	}

	wg.Wait()
	os.Remove(data["scriptfilepath"].(string))
	return ret
}
