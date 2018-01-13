package execute

import (
	"os/exec"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	"github.com/mobingilabs/sesha3/pkg/util"
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

	glog.V(2).Info("run-scp: %v", cmdscp.Args)

	scpb, err := cmdscp.CombinedOutput()
	if err != nil {
		// TODO: should we return err here?
		glog.Errorf("scp failed: %+v", util.ErrV(err))
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

		glog.V(2).Infof("run-ssh: %v", cmdscript.Args)

		scriptout, err := cmdscript.CombinedOutput()
		if err != nil {
			glog.Errorf("ssh exec failed: %+v", util.ErrV(err))
		}

		glog.V(2).Infof("run-ssh out: %v", string(scriptout))

		out.CmdOut = scriptout
		out.Err = err
	}

	return out
}
