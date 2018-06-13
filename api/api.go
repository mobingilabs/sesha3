package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/labstack/echo"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/jwt"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/mobingilabs/sesha3/pkg/constants"
	"github.com/mobingilabs/sesha3/pkg/creds"
	"github.com/mobingilabs/sesha3/pkg/execute"
	"github.com/mobingilabs/sesha3/pkg/metrics"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/mobingilabs/sesha3/pkg/session"
	"github.com/mobingilabs/sesha3/pkg/util"
	"github.com/pkg/errors"
)

type ep struct{}

func New() *ep {
	return &ep{}
}

func (e *ep) elapsed(c echo.Context) {
	fn := c.Get("fnelapsed").(func(echo.Context))
	fn(c)
}

func (e *ep) simpleResponse(c echo.Context, code int, m string) error {
	resp := map[string]string{}
	resp["message"] = m
	return c.JSON(code, resp)
}

func (e *ep) HandleHttpToken(c echo.Context) error {
	defer e.elapsed(c)

	start := time.Now()
	metrics.MetricsTokenRequestCount.Add(1)
	metrics.MetricsCurrentConnection.Add(1)
	defer metrics.MetricsCurrentConnection.Add(-1)
	metrics.MetricsTokenRequest.Add(1)
	defer metrics.MetricsTokenRequest.Add(-1)

	ctx, err := jwt.NewCtx(constants.DATA_DIR)
	if err != nil {
		e.simpleResponse(c, http.StatusUnauthorized, err.Error())
		notify.HookPost(err)

		// if this fails, try force restart to redownload token files
		glog.Exitf("jwt ctx failed: %+v", util.ErrV(err))
	}

	var credentials creds.Credentials

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		glog.Errorf("readall body failed: %+v", util.ErrV(err))
		return err
	}

	defer c.Request().Body.Close()
	glog.V(2).Infof("body (raw): %v", string(body))
	err = json.Unmarshal(body, &credentials)
	if err != nil {
		glog.Errorf("unmarshal failed: %+v", util.ErrV(err))
		return err
	}

	glog.V(2).Infof("body: %+v", credentials)

	ok, err := credentials.Validate()
	if !ok {
		m := "credentials validation failed"
		e.simpleResponse(c, http.StatusInternalServerError, m)
		return errors.New(m)
	}

	if err != nil {
		e.simpleResponse(c, http.StatusInternalServerError, err.Error())
		glog.Errorf("credentials validation failed: %+v", util.ErrV(err))
		return err
	}

	m := make(map[string]interface{})
	m["username"] = credentials.Username
	m["password"] = credentials.Password
	tokenobj, stoken, err := ctx.GenerateToken(m)
	if err != nil {
		glog.Errorf("generate token failed: %+v", util.ErrV(err))
		return err
	}

	end := time.Now()
	metrics.MetricsTokenResponseTime.Set(end.Sub(start).String())

	glog.V(2).Infof("token (obj): %v", tokenobj)
	glog.V(1).Infof("token: %v", stoken)

	reply := make(map[string]string)
	reply["key"] = stoken
	return c.JSON(http.StatusOK, reply)
}

func (e *ep) HandleHttpTtyUrl(c echo.Context) error {
	defer e.elapsed(c)

	start := time.Now()
	metrics.MetricsCurrentConnection.Add(1)
	defer metrics.MetricsCurrentConnection.Add(-1)
	metrics.MetricsConnectionCount.Add(1)
	metrics.MetricsTTYRequest.Add(1)
	defer metrics.MetricsTTYRequest.Add(-1)

	var sess session.Session
	var m map[string]interface{}

	auth := strings.Split(c.Request().Header.Get("Authorization"), " ")
	glog.V(1).Infof("auth header: %v", auth)

	if len(auth) != 2 {
		c.NoContent(http.StatusUnauthorized)
		err := fmt.Errorf("bad authorization")
		glog.Errorf("auth header failed: %+v", util.ErrV(err))
		return err
	}

	ctx, err := jwt.NewCtx(constants.DATA_DIR)
	if err != nil {
		e.simpleResponse(c, http.StatusUnauthorized, err.Error())
		notify.HookPost(err)

		// if this fails, try force restart to redownload token files
		glog.Exitf("jwt ctx failed: %+v", util.ErrV(err))
	}

	btoken := auth[1]
	pt, err := ctx.ParseToken(btoken)
	if err != nil {
		e.simpleResponse(c, http.StatusInternalServerError, err.Error())
		glog.Errorf("parse token failed: %+v", util.ErrV(err))
		return err
	}

	/*
		nc := pt.Claims.(*jwt.WrapperClaims)
		u, _ := nc.Data["username"]
		p, _ := nc.Data["password"]
		glog.V(2).Infof("user: %v", u)

		md5p := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s", p))))
		ok, err := token.ValidateCreds(fmt.Sprintf("%s", u), md5p)
		if !ok {
			m := "check token failed"
			e.simpleResponse(c, http.StatusInternalServerError, m)
			glog.Errorf("%+v", util.ErrV(err, m))
			return errors.New(m)
		}

		if err != nil {
			e.simpleResponse(c, http.StatusInternalServerError, err.Error())
			glog.Errorf("check token failed: %+v", util.ErrV(err))
			return err
		}
	*/

	if !pt.Valid {
		err = fmt.Errorf("invalid token")
		e.simpleResponse(c, http.StatusInternalServerError, err.Error())
		glog.Errorf("invalid token: %+v", util.ErrV(err))
		return err
	}

	glog.V(2).Infof("token: %v", btoken)

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		glog.Errorf("readall body failed: %+v", util.ErrV(err))
		return err
	}

	defer c.Request().Body.Close()
	glog.V(2).Infof("body: %v", string(body))

	err = json.Unmarshal(body, &m)
	if err != nil {
		glog.Errorf("unmarshal failed: %+v", util.ErrV(err))
		return err
	}

	sess.StackId = fmt.Sprintf("%v", m["stackid"])
	sess.User = fmt.Sprintf("%v", m["user"])
	sess.Ip = fmt.Sprintf("%v", m["ip"])
	sess.Timeout = fmt.Sprintf("%v", m["timeout"])
	sess.Timeout = "120"

	flag := fmt.Sprintf("%v", m["flag"])
	pemdir := filepath.Join(constants.DATA_DIR, "pem")
	pemfile := filepath.Join(pemdir, sess.StackId+"-"+flag+".pem")
	sess.PemFile = pemfile

	// create the pem file only if not existent
	if !private.Exists(pemfile) {
		if !private.Exists(pemdir) {
			glog.V(2).Infof("create dir: %v", pemdir)
			err = os.MkdirAll(pemdir, 0700)
			if err != nil {
				glog.Errorf("mkdirall failed: %+v", util.ErrV(err))
				notify.HookPost(err)
				return err
			}
		}

		pemurl := m["pem"].(string)
		glog.V(2).Infof("raw url: %v", pemurl)

		resp, err := http.Get(fmt.Sprintf("%v", pemurl))
		if err != nil {
			glog.Errorf("http get failed: %+v", util.ErrV(err))
			notify.HookPost(err)
			return err
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			glog.Errorf("readall failed: %+v", util.ErrV(err))
			notify.HookPost(err)
			return err
		}

		err = ioutil.WriteFile(pemfile, body, 0600)
		if err != nil {
			glog.Errorf("write file failed: %+v", util.ErrV(err))
			notify.HookPost(err)
			return err
		}
	} else {
		glog.V(1).Infof("reuse: %v", pemfile)
	}

	glog.V(2).Infof("session: %+v", sess)

	// start the ssh session
	randomurl, err := sess.Start()
	if err != nil {
		glog.Errorf("session start failed: %+v", util.ErrV(err))
		notify.HookPost(err)
		return err
	}

	// add this session to our list of running sessions
	session.Sessions.Add(sess)

	var fullurl string

	if params.UseProxy {
		// we don't need randomurl with proxy
	} else {
		if randomurl == "" {
			err := fmt.Errorf("%s", "cannot initialize secure tty access")
			e.simpleResponse(c, http.StatusInternalServerError, err.Error())
			glog.Errorf("session add failed")
			notify.HookPost(err)
			return err
		}

		sess.TtyURL = randomurl
	}

	sess.Online = true
	fullurl = sess.GetFullURL()
	if fullurl == "" {
		err := fmt.Errorf("%s", "cannot initialize secure tty access")
		e.simpleResponse(c, http.StatusInternalServerError, err.Error())
		glog.Errorf("get full url failed")
		notify.HookPost(err)
		return err
	}

	type _url_payload struct {
		Url string `json:"tty_url"`
	}

	end := time.Now()
	metrics.MetricsTTYResponseTime.Set(end.Sub(start).String())

	reply := make(map[string]string)
	reply["tty_url"] = fullurl
	return c.JSON(http.StatusOK, reply)
}

func (e *ep) HandleHttpExec(c echo.Context) error {
	defer e.elapsed(c)

	var in sesha3.ExecScriptPayload

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		glog.Errorf("readall body failed: %+v", util.ErrV(err))
		return err
	}

	defer c.Request().Body.Close()
	glog.V(2).Infof("body (raw): %v", string(body))

	err = json.Unmarshal(body, &in)
	if err != nil {
		glog.Errorf("unmarshal failed: %+v", util.ErrV(err))
		return err
	}

	glog.V(2).Infof("body: %+v", in)

	// token check
	auth := strings.Split(c.Request().Header.Get("Authorization"), " ")
	if len(auth) != 2 {
		c.NoContent(http.StatusUnauthorized)
		err = fmt.Errorf("bad authorization")
		glog.Errorf("auth header failed: %+v", util.ErrV(err))
		return err
	}

	ctx, err := jwt.NewCtx(constants.DATA_DIR)
	if err != nil {
		e.simpleResponse(c, http.StatusUnauthorized, err.Error())
		notify.HookPost(err)

		// if this fails, try force restart to redownload token files
		glog.Exitf("jwt ctx failed: %+v", util.ErrV(err))
	}

	btoken := auth[1]
	pt, err := ctx.ParseToken(btoken)
	if err != nil {
		e.simpleResponse(c, http.StatusInternalServerError, err.Error())
		glog.Errorf("parse token failed: %+v", util.ErrV(err))
		return err
	}

	if !pt.Valid {
		err = fmt.Errorf("invalid token")
		e.simpleResponse(c, http.StatusInternalServerError, err.Error())
		glog.Errorf("invalid token: %+v", util.ErrV(err))
		return err
	}

	glog.V(1).Infof("token: %v", btoken)
	glog.V(1).Infof("pemurl: %v", in.Target.PemUrl)

	// pemfile download for ssh
	resp, err := http.Get(fmt.Sprintf("%v", in.Target.PemUrl))
	if err != nil {
		glog.Errorf("http get failed: %+v", util.ErrV(err))
		notify.HookPost(err)
		return err
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("readall failed: %+v", util.ErrV(err))
		notify.HookPost(err)
		return err
	}

	workdir := filepath.Join(constants.DATA_DIR, in.Target.StackId+"_"+in.Target.Flag)
	glog.V(2).Infof("workdir: %v", workdir)

	if !private.Exists(workdir) {
		err = os.MkdirAll(workdir, 0700)
		if err != nil {
			glog.Errorf("mkdirall failed: %+v", util.ErrV(err))
			notify.HookPost(err)
			return err
		}

		glog.V(2).Infof("workdir created: %v", workdir)
	}

	pemfile := filepath.Join(workdir, in.Target.Ip+".pem")
	glog.V(2).Infof("pemfile: %v", pemfile)

	if !private.Exists(pemfile) {
		err = ioutil.WriteFile(pemfile, body, 0600)
		if err != nil {
			glog.Errorf("write file failed: %+v", util.ErrV(err))
			notify.HookPost(err)
			return err
		}

		glog.V(2).Infof("pemfile created: %v", pemfile)
	}

	// write script to temporary file
	script := filepath.Join(workdir, "_runscript")
	err = ioutil.WriteFile(script, in.Script, 0755)
	if err != nil {
		glog.Errorf("write file failed: %+v", util.ErrV(err))
		notify.HookPost(err)
		return err
	}

	err = os.Chmod(script, 0755)
	glog.V(2).Infof("script: %v", script)

	if err != nil {
		glog.Errorf("chmod failed: %+v", util.ErrV(err))
		notify.HookPost(err)
		return err
	}

	glog.V(2).Infof("script created: %v", script)

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

	return c.JSON(http.StatusOK, sout)
}
