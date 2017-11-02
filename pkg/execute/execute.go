package execute

import (
	"os"
	"os/exec"

	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/pkg/errors"
)

func SshCmd(stackid string, data map[string]interface{}) ([]sesha3.ExecScriptInstanceResponse, error) {
	ips := data["target"].([]string)
	pemfile := data["pem"].(string)
	ret := []sesha3.ExecScriptInstanceResponse{}
	d.Info("exec:", ips)

	for _, ip := range ips {
		var out sesha3.ExecScriptInstanceResponse
		out.Ip = ip
		out.StackId = stackid
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
				return nil, errors.Wrap(err, "ssh run script failed")
			}

			out.CmdOut = scriptout
			ret = append(ret, out)
		}
	}

	os.Remove(data["scriptfilepath"].(string))
	return ret, nil
}
