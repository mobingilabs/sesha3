package api

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/jwt"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/mobingilabs/sesha3/pkg/metrics"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/mobingilabs/sesha3/pkg/session"
	"github.com/mobingilabs/sesha3/pkg/token"
	"github.com/pkg/errors"
)

func handleHttpTtyUrl(c *ApiController) {
	start := time.Now()
	metrics.MetricsCurrentConnection.Add(1)
	defer metrics.MetricsCurrentConnection.Add(-1)
	metrics.MetricsConnectionCount.Add(1)
	metrics.MetricsTTYRequest.Add(1)
	defer metrics.MetricsTTYRequest.Add(-1)

	var sess session.Session
	var m map[string]interface{}

	auth := strings.Split(c.Ctx.Request.Header.Get("Authorization"), " ")
	c.info("auth-hdr:", auth)
	if len(auth) != 2 {
		c.Ctx.ResponseWriter.WriteHeader(401)
		d.Error("auth header failed")
		return
	}

	ctx, err := jwt.NewCtx()
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		d.Error(errors.Wrap(err, "jwt ctx failed"))
		return
	}

	btoken := auth[1]
	pt, err := ctx.ParseToken(btoken)
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		d.Error(errors.Wrap(err, "parse token failed"))
		return
	}

	nc := pt.Claims.(*jwt.WrapperClaims)
	u, _ := nc.Data["username"]
	p, _ := nc.Data["password"]
	d.Info("user:", u)

	md5p := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s", p))))
	ok, err := token.CheckToken(fmt.Sprintf("%s", u), md5p)
	if !ok {
		c.Ctx.ResponseWriter.WriteHeader(401)
		d.Error(errors.Wrap(err, "check token not ok"))
		return
	}

	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		d.Error(errors.Wrap(err, "check token failed"))
		return
	}

	d.Info("token:", btoken)
	d.Info("body:", string(c.Ctx.Input.RequestBody))
	err = json.Unmarshal(c.Ctx.Input.RequestBody, &m)
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		d.Error(errors.Wrap(err, "unmarshal body failed"))
		notify.HookPost(err)
		return
	}

	sess.StackId = fmt.Sprintf("%v", m["stackid"])
	sess.User = fmt.Sprintf("%v", m["user"])
	sess.Ip = fmt.Sprintf("%v", m["ip"])
	sess.Timeout = fmt.Sprintf("%v", m["timeout"])
	sess.Timeout = "120"

	flag := fmt.Sprintf("%v", m["flag"])
	pemdir := os.TempDir() + "/sesha3/pem/"
	pemfile := pemdir + sess.StackId + "-" + flag + ".pem"
	sess.PemFile = pemfile

	// create the pem file only if not existent
	if !private.Exists(pemfile) {
		if !private.Exists(pemdir) {
			d.Info("create", pemdir)
			err = os.MkdirAll(pemdir, 0700)
			if err != nil {
				c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
				notify.HookPost(err)
				return
			}
		}

		pemurl := m["pem"].(string)
		d.Info("rawurl:", pemurl)
		resp, err := http.Get(fmt.Sprintf("%v", pemurl))
		if err != nil {
			c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
			d.Error(errors.Wrap(err, "read body failed"))
			notify.HookPost(err)
			return
		}

		err = ioutil.WriteFile(pemfile, body, 0600)
		if err != nil {
			c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
			d.Error(errors.Wrap(err, "write pem failed"))
			notify.HookPost(err)
			return
		}
	} else {
		d.Info("reuse:", pemfile)
	}

	// start the ssh session
	randomurl, err := sess.Start()
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		d.Error(errors.Wrap(err, "session start failed"))
		notify.HookPost(err)
		return
	}

	// add this session to our list of running sessions
	session.Sessions.Add(sess)
	if randomurl == "" {
		err := fmt.Errorf("%s", "cannot initialize secure tty access")
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		d.Error(errors.Wrap(err, "session add failed"))
		notify.HookPost(err)
		return
	} else {
		sess.Online = true
	}

	var fullurl string
	sess.TtyURL = randomurl
	fullurl = sess.GetFullURL()
	if fullurl == "" {
		err := fmt.Errorf("%s", "cannot initialize secure tty access")
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		d.Error(errors.Wrap(err, "get full url failed"))
		notify.HookPost(err)
		return
	}

	type _url_payload struct {
		Url string `json:"tty_url"`
	}

	end := time.Now()
	metrics.MetricsTTYResponseTime.Set(end.Sub(start).String())

	reply := make(map[string]string)
	reply["tty_url"] = fullurl
	c.Data["json"] = reply
	c.ServeJSON()
}
