package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"log/syslog"
	"net/http"
	"time"

	"github.com/astaxie/beego"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/jwt"
	"github.com/mobingilabs/sesha3/api"
	"github.com/mobingilabs/sesha3/pkg/cert"
	"github.com/mobingilabs/sesha3/pkg/metrics"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/mobingilabs/sesha3/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ServeCtx struct {
	localUrl string
}

var logger *syslog.Writer

func ServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "run as server",
		Long:  `Run as server.`,
		Run: func(cmd *cobra.Command, args []string) {
			if params.UseSyslog {
				logger, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, "sesha3")
				if err != nil {
					notify.HookPost(errors.Wrap(err, "syslog setup failed, fatal"))
					d.ErrorTraceExit(err, 1)
				}

				log.SetFlags(0)
				log.SetPrefix("[" + util.GetEc2Id() + "] ")
				log.SetOutput(logger)
			}

			metrics.MetricsType.MetricsInit()
			eps, _ := cmd.Flags().GetStringArray("notify-endpoints")
			err := notify.Notifier.Init(eps)
			if err != nil {
				d.Error(err)
			}

			d.Info("--- server start ---")
			d.Info("dns:", util.GetPublicDns()+":"+params.Port)
			d.Info("ec2:", params.Ec2Id)
			d.Info("syslog:", params.UseSyslog)
			d.Info("region:", params.Region)
			d.Info("credprof:", params.CredProfile)

			// try setting up LetsEncrypt certificates locally
			err = cert.SetupLetsEncryptCert(true)
			if err != nil {
				notify.HookPost(err)
				d.Error(err)
			} else {
				certfolder := "/etc/letsencrypt/live/" + util.Domain()
				d.Info("certificate folder:", certfolder)
			}

			startm := "--- server start ---\n"
			startm += "dns: " + util.GetPublicDns() + "\n"
			startm += "ec2: " + params.Ec2Id + "\n"
			startm += "syslog: " + fmt.Sprintf("%v", params.UseSyslog)
			notify.HookPost(startm)

			beego.BConfig.ServerName = "sesha3:1.0.0"
			beego.BConfig.RunMode = beego.PROD
			if params.IsDev {
				beego.BConfig.RunMode = beego.DEV
			}

			// needed for http input body in request to be available for non-get and head reqs
			beego.BConfig.CopyRequestBody = true

			beego.Router("/", &api.ApiController{}, "get:DispatchRoot")
			beego.Router("/scratch", &api.ApiController{}, "get:DispatchScratch")
			beego.Router("/token", &api.ApiController{}, "post:DispatchToken")
			beego.Router("/ttyurl", &api.ApiController{}, "post:DispatchTtyUrl")
			beego.Router("/exec", &api.ApiController{}, "post:DispatchExec")
			beego.Run(":" + params.Port)

			/*
				router := mux.NewRouter()
				router.HandleFunc("/", handleHttpRoot(c)).Methods(http.MethodGet)
				router.HandleFunc("/version", handleHttpVersion(c)).Methods(http.MethodGet)
				router.HandleFunc("/token", handleHttpToken(c)).Methods(http.MethodGet)
				router.HandleFunc("/ttyurl", handleHttpPtyUrl(c)).Methods(http.MethodGet)
				router.HandleFunc("/exec", handleHttpExecScript(c)).Methods(http.MethodGet)
				router.Handle("/debug/vars", metrics.MetricsHandler)

				// start our http server
				err = http.ListenAndServe(":"+params.Port, router)
				if err != nil {
					notify.HookPost(errors.Wrap(err, "server failed, fatal"))
					d.ErrorTraceExit(err, 1)
				}
			*/
		},
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&params.Port, "port", "8080", "server port")
	return cmd
}

func handleHttpRoot(c *ServeCtx) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Copyright (c) Mobingi. All rights reserved."))
	})
}

func handleHttpToken(c *ServeCtx) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		metrics.MetricsTokenRequestCount.Add(1)
		metrics.MetricsCurrentConnection.Add(1)
		defer metrics.MetricsCurrentConnection.Add(-1)
		metrics.MetricsTokenRequest.Add(1)
		defer metrics.MetricsTokenRequest.Add(-1)

		ctx, err := jwt.NewCtx()
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		type creds_t struct {
			Username string `json:"username"`
			Password string `json:"passwd"`
		}

		var up creds_t
		err = json.Unmarshal(body, &up)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		m := make(map[string]interface{})
		m["username"] = up.Username
		m["password"] = up.Password
		_, stoken, err := ctx.GenerateToken(m)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		d.Info("generated token:", stoken)
		payload := `{"key":"` + stoken + `"}`
		w.Write([]byte(payload))
		end := time.Now()
		metrics.MetricsTokenResponseTime.Set(end.Sub(start).String())
	})
}

/*
func handleHttpPtyUrl(c *ServeCtx) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		metrics.MetricsCurrentConnection.Add(1)
		defer metrics.MetricsCurrentConnection.Add(-1)
		metrics.MetricsConnectionCount.Add(1)
		metrics.MetricsTTYRequest.Add(1)
		defer metrics.MetricsTTYRequest.Add(-1)

		var sess session.Session
		var m map[string]interface{}

		auth := strings.Split(r.Header.Get("Authorization"), " ")
		if len(auth) != 2 {
			w.WriteHeader(401)
			return
		}

		ctx, err := jwt.NewCtx()
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		btoken := auth[1]
		pt, err := ctx.ParseToken(btoken)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		nc := pt.Claims.(*jwt.WrapperClaims)
		u, _ := nc.Data["username"]
		p, _ := nc.Data["password"]
		d.Info("user:", u)

		md5p := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s", p))))
		ok, err := token.CheckToken(params.CredProfile, params.Region, fmt.Sprintf("%s", u), md5p)
		if !ok {
			w.WriteHeader(401)
			return
		}

		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		d.Info("token:", btoken)
		d.Info("body:", string(body))
		err = json.Unmarshal(body, &m)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		pemdir := os.TempDir() + "/sesha3/pem/"
		pemfile := pemdir + sess.StackId + ".pem"
		if !private.Exists(pemfile) {
			// create the pem directory if not exists
			if !private.Exists(pemdir) {
				d.Info("create", pemdir)
				err = os.MkdirAll(pemdir, 0700)
				if err != nil {
					w.Write(sesha3.NewSimpleError(err).Marshal())
					notify.HookPost(err)
					return
				}
			}

			pemurl := m["pem"].(string)
			d.Info("rawurl:", pemurl)
			resp, err := http.Get(fmt.Sprintf("%v", pemurl))
			if err != nil {
				w.Write(sesha3.NewSimpleError(err).Marshal())
				notify.HookPost(err)
				return
			}

			defer resp.Body.Close()
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				w.Write(sesha3.NewSimpleError(err).Marshal())
				notify.HookPost(err)
				return
			}

			err = ioutil.WriteFile(pemfile, body, 0600)
			if err != nil {
				w.Write(sesha3.NewSimpleError(err).Marshal())
				notify.HookPost(err)
				return
			}
		} else {
			d.Info("reuse:", pemfile)
		}

		sess.User = fmt.Sprintf("%v", m["user"])
		sess.Ip = fmt.Sprintf("%v", m["ip"])
		sess.StackId = fmt.Sprintf("%v", m["stackid"])
		sess.Timeout = fmt.Sprintf("%v", m["timeout"])

		sess.PemFile = pemfile
		randomurl, err := sess.Start()
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		// add this session to our list of running sessions
		session.Sessions.Add(sess)
		if randomurl == "" {
			err := fmt.Errorf("%s", "cannot initialize secure tty access")
			w.Write(sesha3.NewSimpleError(err).Marshal())
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
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		payload := `{"tty_url":"` + fullurl + `"}`
		w.Write([]byte(payload))
		end := time.Now()
		metrics.MetricsTTYResponseTime.Set(end.Sub(start).String())
	})
}

func handleHttpExecScript(c *ServeCtx) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in sesha3.ExecScriptPayload

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		d.Info("body:", string(body))
		err = json.Unmarshal(body, &in)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		// token check
		auth := strings.Split(r.Header.Get("Authorization"), " ")
		if len(auth) != 2 {
			w.WriteHeader(401)
			return
		}

		ctx, err := jwt.NewCtx()
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		btoken := auth[1]
		pt, err := ctx.ParseToken(btoken)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		nc := pt.Claims.(*jwt.WrapperClaims)
		u, _ := nc.Data["username"]
		p, _ := nc.Data["password"]
		d.Info("user:", u)
		md5p := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s", p))))
		ok, err := token.CheckToken(params.CredProfile, params.Region, fmt.Sprintf("%s", u), md5p)
		if !ok {
			w.WriteHeader(401)
			return
		}

		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		d.Info("token:", btoken)

		// pemfile download for ssh
		d.Info("pemurl:", in.Target.PemUrl)
		resp, err := http.Get(fmt.Sprintf("%v", in.Target.PemUrl))
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		workdir := os.TempDir() + "/" + in.Target.StackId + "_" + in.Target.Flag + "/"
		d.Info("workdir:", workdir)
		if !private.Exists(workdir) {
			err = os.MkdirAll(workdir, 0700)
			if err != nil {
				w.Write(sesha3.NewSimpleError(err).Marshal())
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
				w.Write(sesha3.NewSimpleError(err).Marshal())
				notify.HookPost(err)
				return
			}

			d.Info("pemfile created:", pemfile)
		}

		// write script to temporary file
		script := workdir + "_runscript"
		err = ioutil.WriteFile(script, in.Script, 0755)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		err = os.Chmod(script, 0755)
		d.Info("script:", script)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
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

		payload, err := json.Marshal(sout)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
		}

		d.Info("reply:", string(payload))
		w.Write(payload)
	})
}
*/

/*
func describeSessions(w http.ResponseWriter, req *http.Request) {
	ds := session.Sessions.Describe()
	b, err := json.Marshal(ds)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}

	w.Write(b)
}
*/

func handleHttpVersion(c *ServeCtx) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics.MetricsCurrentConnection.Add(1)
		defer metrics.MetricsCurrentConnection.Add(-1)
		w.Write([]byte(`{"version":"v0.0.15-dev"}`))
	})
}
