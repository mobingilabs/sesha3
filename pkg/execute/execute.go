package execute

import (
	"os/exec"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
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

	glog.Info("run-scp: %v", cmdscp.Args)

	scpb, err := cmdscp.CombinedOutput()
	if err != nil {
		// TODO: should we return err here?
		glog.Errorf("scp failed: %v", err)
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

		glog.Infof("run-ssh: %v", cmdscript.Args)

		scriptout, err := cmdscript.CombinedOutput()
		if err != nil {
			glog.Errorf("ssh exec failed: %v", err)
		}

		out.CmdOut = scriptout
		out.Err = err
	}

	return out
}
