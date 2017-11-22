package execute

import (
	"os/exec"
	"path/filepath"

	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/pkg/errors"
)

type SshCmdInput struct {
	Ip     string
	Pem    string
	Script string
	VmUser string
}

func SshCmd(in SshCmdInput) sesha3.ExecScriptInstanceResponse {
	out := sesha3.ExecScriptInstanceResponse{Ip: in.Ip}
	cmdscp := exec.Command(
		"/usr/bin/scp",
		"-p",
		"-i", in.Pem,
		"-o", "StrictHostKeyChecking=no",
		in.Script,
		in.VmUser+"@"+in.Ip+":/tmp/",
	)

	d.Info("run-scp:", cmdscp.Args)
	scpb, err := cmdscp.CombinedOutput()
	if err != nil {
		// TODO: should we return err here?
		d.Error(errors.Wrap(err, "scp failed"))
		out.CmdOut = scpb
		out.Err = err
	} else {
		cmdscript := exec.Command(
			"/usr/bin/ssh",
			"-o",
			"StrictHostKeyChecking=no",
			"-i", in.Pem,
			in.VmUser+"@"+in.Ip,
			"/tmp/"+filepath.Base(in.Script),
		)

		d.Info("run-ssh:", cmdscript.Args)
		scriptout, err := cmdscript.CombinedOutput()
		if err != nil {
			d.Error(errors.Wrap(err, "ssh exec failed"))
		}

		out.CmdOut = scriptout
		out.Err = err
	}

	return out
}
