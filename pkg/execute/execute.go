package execute

import (
	"os"
	"os/exec"
	"regexp"
	"strings"

	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
)

type Result struct {
	Stackid string `json:"stack_id"`
	Ip      string `json:"ip"`
	Out     string `json:"out"`
}

func Sshcmd(stackid string, data map[string]interface{}) []Result {
	ips := data["target"].([]string)
	pemfile := data["pem"].(string)
	ret := []Result{}
	d.Info("exec:", ips)

	rep := regexp.MustCompile(`^\n|^\r|\n$|\r$`)
	for _, ip := range ips {
		var out Result
		out.Ip = ip
		out.Stackid = stackid
		cmdscp := exec.Command(
			"/usr/bin/scp",
			"-p",
			"-i", pemfile,
			"-o", "StrictHostKeyChecking=no",
			data["scriptfilepath"].(string),
			data["user"].(string)+"@"+ip+":/tmp/",
		)

		d.Info("run-scp:", cmdscp.Args)
		scpb, err := cmdscp.CombinedOutput()
		if err != nil {
			out.Out = rep.ReplaceAllString(string(scpb), "")
			ret = append(ret, out)
		} else {
			cmdscript := exec.Command(
				"/usr/bin/ssh",
				"-o",
				"StrictHostKeyChecking=no",
				"-i", pemfile,
				data["user"].(string)+"@"+ip,
				"/tmp/"+data["script_name"].(string),
			)

			d.Info("run-ssh:", cmdscript.Args)
			scriptout, err := cmdscript.CombinedOutput()
			if err != nil {
				d.Error("script:", err)
			}

			out.Out = rep.ReplaceAllString(strings.Replace(string(scriptout), "\r", "\n", -1), "")
			// ste := strings.Split(strings.Replace(scripterr, "\r", "\n", -1), "\n")
			// out.Stderr = rep.ReplaceAllString(strings.Join(ste[0:len(ste)-1], "\n"), "")
			d.Info("out:", out.Out)
			ret = append(ret, out)
		}
	}

	os.Remove(data["scriptfilepath"].(string))
	return ret
}
