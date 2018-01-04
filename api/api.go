package api

import (
	"crypto/md5"
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
	"github.com/mobingilabs/sesha3/pkg/execute"
	"github.com/mobingilabs/sesha3/pkg/metrics"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/mobingilabs/sesha3/pkg/session"
	"github.com/mobingilabs/sesha3/pkg/token"
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

type credentials struct {
	Username string `json:"username"`
	Password string `json:"passwd"`
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
		notify.HookPost(err)
		e.simpleResponse(c, http.StatusUnauthorized, err.Error())

		// if this fails, try force restart to redownload token files
		glog.Exitf("jwt ctx failed: %v", err)
	}

	var creds credentials

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		glog.Errorf("readall body failed: %v", err)
		return err
	}

	defer c.Request().Body.Close()
	glog.Infof("body (raw): %v", string(body))
	err = json.Unmarshal(body, &creds)
	if err != nil {
		glog.Errorf("unmarshal failed: %v", err)
		return err
	}

	glog.Infof("body: %+v", creds)

	m := make(map[string]interface{})
	m["username"] = creds.Username
	m["password"] = creds.Password
	tokenobj, stoken, err := ctx.GenerateToken(m)
	if err != nil {
		glog.Errorf("generate token failed: %v", err)
		return err
	}

	end := time.Now()
	metrics.MetricsTokenResponseTime.Set(end.Sub(start).String())

	glog.Infof("token (obj): %v", tokenobj)
	glog.Infof("token: %v", stoken)

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
	glog.Infof("auth header: %v", auth)
	if len(auth) != 2 {
		c.NoContent(http.StatusUnauthorized)
		err := errors.New("bad authorization")
		glog.Errorf("auth header failed: %v", err)
		return err
	}

	ctx, err := jwt.NewCtx(constants.DATA_DIR)
	if err != nil {
		notify.HookPost(err)
		e.simpleResponse(c, http.StatusUnauthorized, err.Error())

		// if this fails, try force restart to redownload token files
		glog.Exitf("jwt ctx failed: %v", err)
	}

	btoken := auth[1]
	pt, err := ctx.ParseToken(btoken)
	if err != nil {
		e.simpleResponse(c, http.StatusInternalServerError, err.Error())
		glog.Errorf("parse token failed: %v", err)
		return err
	}

	nc := pt.Claims.(*jwt.WrapperClaims)
	u, _ := nc.Data["username"]
	p, _ := nc.Data["password"]
	glog.Infof("user: %v", u)

	md5p := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s", p))))
	ok, err := token.CheckToken(fmt.Sprintf("%s", u), md5p)
	if !ok {
		m := "check token failed"
		e.simpleResponse(c, http.StatusInternalServerError, m)
		glog.Errorf(m)
		return errors.New(m)
	}

	if err != nil {
		e.simpleResponse(c, http.StatusInternalServerError, err.Error())
		glog.Errorf("check token failed: %v", err)
		return err
	}

	glog.Infof("token: %v", btoken)

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		glog.Errorf("readall body failed: %v", err)
		return err
	}

	defer c.Request().Body.Close()
	glog.Infof("body: %v", string(body))

	err = json.Unmarshal(body, &m)
	if err != nil {
		glog.Errorf("unmarshal failed: %v", err)
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
			glog.Infof("create dir: %v", pemdir)
			err = os.MkdirAll(pemdir, 0700)
			if err != nil {
				glog.Errorf("mkdirall failed: %v", err)
				notify.HookPost(err)
				return err
			}
		}

		pemurl := m["pem"].(string)
		glog.Infof("raw url: %v", pemurl)

		resp, err := http.Get(fmt.Sprintf("%v", pemurl))
		if err != nil {
			glog.Errorf("http get failed: %v", err)
			notify.HookPost(err)
			return err
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			glog.Errorf("readall failed: %v", err)
			notify.HookPost(err)
			return err
		}

		err = ioutil.WriteFile(pemfile, body, 0600)
		if err != nil {
			glog.Errorf("write file failed: %v", err)
			notify.HookPost(err)
			return err
		}
	} else {
		glog.Infof("reuse: %v", pemfile)
	}

	// start the ssh session
	randomurl, err := sess.Start()
	if err != nil {
		glog.Errorf("session start failed: %v", err)
		notify.HookPost(err)
		return err
	}

	// add this session to our list of running sessions
	session.Sessions.Add(sess)
	if randomurl == "" {
		err := fmt.Errorf("%s", "cannot initialize secure tty access")
		e.simpleResponse(c, http.StatusInternalServerError, err.Error())
		glog.Errorf("session add failed")
		notify.HookPost(err)
		return err
	} else {
		sess.Online = true
	}

	var fullurl string
	sess.TtyURL = randomurl
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
		glog.Errorf("readall body failed: %v", err)
		return err
	}

	defer c.Request().Body.Close()
	glog.Infof("body (raw): %v", string(body))

	err = json.Unmarshal(body, &in)
	if err != nil {
		glog.Errorf("unmarshal failed: %v", err)
		return err
	}

	glog.Infof("body: %+v", in)

	// token check
	auth := strings.Split(c.Request().Header.Get("Authorization"), " ")
	if len(auth) != 2 {
		c.NoContent(http.StatusUnauthorized)
		err := errors.New("bad authorization")
		glog.Errorf("auth header failed: %v", err)
		return err
	}

	ctx, err := jwt.NewCtx(constants.DATA_DIR)
	if err != nil {
		notify.HookPost(err)
		e.simpleResponse(c, http.StatusUnauthorized, err.Error())

		// if this fails, try force restart to redownload token files
		glog.Exitf("jwt ctx failed: %v", err)
	}

	btoken := auth[1]
	pt, err := ctx.ParseToken(btoken)
	if err != nil {
		e.simpleResponse(c, http.StatusInternalServerError, err.Error())
		glog.Errorf("parse token failed: %v", err)
		return err
	}

	nc := pt.Claims.(*jwt.WrapperClaims)
	u, _ := nc.Data["username"]
	p, _ := nc.Data["password"]
	glog.Infof("user: %v", u)

	md5p := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s", p))))
	ok, err := token.CheckToken(fmt.Sprintf("%s", u), md5p)
	if !ok {
		m := "check token failed"
		e.simpleResponse(c, http.StatusUnauthorized, m)
		glog.Errorf(m)
		return errors.New(m)
	}

	if err != nil {
		e.simpleResponse(c, http.StatusUnauthorized, err.Error())
		glog.Errorf("check token failed: %v", err)
		return err
	}

	glog.Infof("token: %v", btoken)
	glog.Infof("pemurl: %v", in.Target.PemUrl)

	// pemfile download for ssh
	resp, err := http.Get(fmt.Sprintf("%v", in.Target.PemUrl))
	if err != nil {
		glog.Errorf("http get failed: %v", err)
		notify.HookPost(err)
		return err
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("readall failed: %v", err)
		notify.HookPost(err)
		return err
	}

	workdir := filepath.Join(constants.DATA_DIR, in.Target.StackId+"_"+in.Target.Flag)
	glog.Infof("workdir: %v", workdir)

	if !private.Exists(workdir) {
		err = os.MkdirAll(workdir, 0700)
		if err != nil {
			glog.Errorf("mkdirall failed: %v", err)
			notify.HookPost(err)
			return err
		}

		glog.Infof("workdir created: %v", workdir)
	}

	pemfile := filepath.Join(workdir, in.Target.Ip+".pem")
	glog.Infof("pemfile: %v", pemfile)

	if !private.Exists(pemfile) {
		err = ioutil.WriteFile(pemfile, body, 0600)
		if err != nil {
			glog.Errorf("write file failed: %v", err)
			notify.HookPost(err)
			return err
		}

		glog.Infof("pemfile created: %v", pemfile)
	}

	// write script to temporary file
	script := filepath.Join(workdir, "_runscript")
	err = ioutil.WriteFile(script, in.Script, 0755)
	if err != nil {
		glog.Errorf("write file failed: %v", err)
		notify.HookPost(err)
		return err
	}

	err = os.Chmod(script, 0755)
	glog.Infof("script: %v", script)

	if err != nil {
		glog.Errorf("chmod failed: %v", err)
		notify.HookPost(err)
		return err
	}

	glog.Infof("script created: %v", script)

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

/*
type ApiController struct {
	beego.Controller
	noAuth    bool
	sessionId string
}

func (c *ApiController) Prepare() {
	if c.sessionId == "" {
		c.sessionId = fmt.Sprintf("{%s}", uuid.NewV4())
	}

	// do auth by default
	if !c.noAuth {
		c.info("base auth:")
	}

	c.info("remote:", c.Ctx.Request.RemoteAddr)
	c.info("urlpath:", c.Ctx.Request.URL.Path)
}

// info is our local info logger with session id as prefix.
func (c *ApiController) info(v ...interface{}) {
	m := fmt.Sprintln(v...)
	log.Print(fmt.Sprintf("s-%s:info: %s", c.sessionId, m))
}

// info is our local error logger with session id as prefix.
func (c *ApiController) err(v ...interface{}) {
	m := fmt.Sprintln(v...)
	log.Print(fmt.Sprintf("s-%s:error: %s", c.sessionId, m))
}

func (c *ApiController) getClientIp() string {
	s := strings.Split(c.Ctx.Request.RemoteAddr, ":")
	if len(s) == 0 {
		return "?"
	}

	return s[0]
}

func (c *ApiController) DispatchScratch() {
	type x struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	c.info("hello info")
	c.err("hello error")
	c.Data["json"] = x{Name: "foo", Value: "bar"}
	c.ServeJSON()
}

func (c *ApiController) DispatchRoot() {
	c.Ctx.ResponseWriter.Write([]byte("Copyright (c) Mobingi. All rights reserved."))
}

func (c *ApiController) DispatchToken() {
	handleHttpToken(c)
}

func (c *ApiController) DispatchTtyUrl() {
	handleHttpTtyUrl(c)
}

func (c *ApiController) DispatchExec() {
	handleHttpExec(c)
}
*/
