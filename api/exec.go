package api

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/jwt"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/mobingilabs/sesha3/pkg/execute"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/mobingilabs/sesha3/pkg/token"
)

func handleHttpExec(c *ApiController) {
	var in sesha3.ExecScriptPayload

	d.Info("body:", string(c.Ctx.Input.RequestBody))
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &in)
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}

	// token check
	auth := strings.Split(c.Ctx.Request.Header.Get("Authorization"), " ")
	if len(auth) != 2 {
		c.Ctx.ResponseWriter.WriteHeader(401)
		return
	}

	ctx, err := jwt.NewCtx()
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	btoken := auth[1]
	pt, err := ctx.ParseToken(btoken)
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	nc := pt.Claims.(*jwt.WrapperClaims)
	u, _ := nc.Data["username"]
	p, _ := nc.Data["password"]
	d.Info("user:", u)
	md5p := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s", p))))
	ok, err := token.CheckToken(params.CredProfile, params.Region, fmt.Sprintf("%s", u), md5p)
	if !ok {
		c.Ctx.ResponseWriter.WriteHeader(401)
		return
	}

	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	d.Info("token:", btoken)

	// pemfile download for ssh
	d.Info("pemurl:", in.Target.PemUrl)
	resp, err := http.Get(fmt.Sprintf("%v", in.Target.PemUrl))
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}

	workdir := os.TempDir() + "/" + in.Target.StackId + "_" + in.Target.Flag + "/"
	d.Info("workdir:", workdir)
	if !private.Exists(workdir) {
		err = os.MkdirAll(workdir, 0700)
		if err != nil {
			c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		d.Info("workdir created:", workdir)
	}

	pemfile := workdir + in.Target.Ip + ".pem"
	d.Info("pemfile:", pemfile)
	if !private.Exists(pemfile) {
		err = ioutil.WriteFile(pemfile, body, 0600)
		if err != nil {
			c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		d.Info("pemfile created:", pemfile)
	}

	// write script to temporary file
	script := workdir + "_runscript"
	err = ioutil.WriteFile(script, in.Script, 0755)
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}

	err = os.Chmod(script, 0755)
	d.Info("script:", script)
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}

	d.Info("script created:", script)

	// actual script execution
	out := execute.SshCmd(execute.SshCmdInput{
		Ip:     in.Target.Ip,
		Pem:    pemfile,
		Script: script,
		VmUser: in.Target.VmUser,
	})

	sout := sesha3.ExecScriptStackResponse{
		StackId: in.Target.StackId,
		Outputs: []sesha3.ExecScriptInstanceResponse{out},
	}

	c.Data["json"] = sout
	c.ServeJSON()
}
