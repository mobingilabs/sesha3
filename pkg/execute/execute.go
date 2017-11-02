package execute

import (
	"os"
	"os/exec"

	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
)

func SshCmd(data map[string]interface{}) []sesha3.ExecScriptInstanceResponse {
	ips := data["target"].([]string)
	pemfile := data["pem"].(string)
	ret := []sesha3.ExecScriptInstanceResponse{}
	d.Info("exec:", ips)
	for _, ip := range ips {
		var out sesha3.ExecScriptInstanceResponse
		out.Ip = ip
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
			// TODO: should we return err here?
			out.CmdOut = scpb
			out.Err = err
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
			out.CmdOut = scriptout
			out.Err = err
			ret = append(ret, out)
		}
	}

	os.Remove(data["scriptfilepath"].(string))
	return ret
}
