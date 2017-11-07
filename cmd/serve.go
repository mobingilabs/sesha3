package cmd

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"log/syslog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/jwt"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/mobingilabs/sesha3/pkg/awsports"
	"github.com/mobingilabs/sesha3/pkg/execute"
	"github.com/mobingilabs/sesha3/pkg/metrics"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/mobingilabs/sesha3/pkg/session"
	"github.com/mobingilabs/sesha3/pkg/token"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var logger *syslog.Writer

func ServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "run as server",
		Long:  `Run as server.`,
		Run:   serve,
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&params.Domain, "domain", "sesha3.labs.mobingi.com", "server domain")
	cmd.Flags().StringVar(&params.Port, "port", "443", "server port")
	return cmd
}

func serve(cmd *cobra.Command, args []string) {
	if params.UseSyslog {
		logger, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, "sesha3")
		if err != nil {
			notify.HookPost(errors.Wrap(err, "syslog setup failed, fatal"))
			d.ErrorTraceExit(err, 1)
		}

		log.SetFlags(0)
		log.SetOutput(logger)
	}

	metrics.MetricsType.MetricsInit()
	eps, _ := cmd.Flags().GetStringArray("notify-endpoints")
	err := notify.Notifier.Init(eps)
	if err != nil {
		d.Error(err)
	}

	d.Info("--- server start ---")
	d.Info("url:", params.Domain+":"+params.Port)
	d.Info("syslog:", params.UseSyslog)
	d.Info("region:", params.Region)
	d.Info("server ec2:", params.Ec2Id)
	d.Info("credprof:", params.CredProfile)
	srcdir := cmdline.Dir()
	d.Info("srcdir:", srcdir)
	if !private.Exists(srcdir + "/certs") {
		err := os.MkdirAll(srcdir+"/certs", os.ModePerm)
		notify.HookPost(errors.Wrap(err, "create certs folder failed (fatal)"))
	}

	err = awsports.Download(params.Environment, params.Region, params.CredProfile)
	if err != nil {
		notify.HookPost(errors.Wrap(err, "server failed, fatal"))
		d.ErrorTraceExit(err, 1)
	}

	// redirect every http request to https
	// go http.ListenAndServe(":80", http.HandlerFunc(redirect))
	// everything else will be https

	startm := "--- server start ---\n"
	startm += "url: " + params.Domain + "\n"
	startm += "syslog: " + fmt.Sprintf("%v", params.UseSyslog) + "\n"
	startm += "region: " + params.Region + "\n"
	startm += "server ec2: " + params.Ec2Id + "\n"
	startm += "credprofile: " + params.CredProfile
	notify.HookPost(startm)

	certfolder := cmdline.Dir() + "/certs"
	router := mux.NewRouter()
	router.HandleFunc("/token", generateToken).Methods(http.MethodGet)
	router.HandleFunc("/ttyurl", ttyurl).Methods(http.MethodGet)
	// router.HandleFunc("/sessions", describeSessions).Methods(http.MethodGet)
	router.HandleFunc("/version", version).Methods(http.MethodGet)
	router.HandleFunc("/exec", execScript).Methods(http.MethodGet)
	// https://sesha3.labs.mobingi.com/debug/vars
	router.Handle("/debug/vars", metrics.MetricsHandler)
	err = http.ListenAndServeTLS(":"+params.Port,
		certfolder+"/fullchain.pem",
		certfolder+"/privkey.pem",
		router)

	if err != nil {
		notify.HookPost(errors.Wrap(err, "server failed, fatal"))
		d.ErrorTraceExit(err, 1)
	}
}

func generateToken(w http.ResponseWriter, r *http.Request) {
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
}

func ttyurl(w http.ResponseWriter, r *http.Request) {
	//metrics
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

	sess.User = fmt.Sprintf("%v", m["user"])
	sess.Ip = fmt.Sprintf("%v", m["ip"])
	sess.StackId = fmt.Sprintf("%v", m["stackid"])
	sess.Timeout = fmt.Sprintf("%v", m["timeout"])
	pemdir := os.TempDir() + "/user/"
	if !private.Exists(pemdir) {
		d.Info("create", pemdir)
		err = os.MkdirAll(pemdir, 0700)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}
	}

	pemfile := os.TempDir() + "/user/" + sess.StackId + ".pem"
	err = ioutil.WriteFile(pemfile, body, 0600)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}

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
}

func execScript(w http.ResponseWriter, r *http.Request) {
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

	d.Info("out:", string(out.CmdOut))

	/*
		// execute cmd
		results := []sesha3.ExecScriptStackResponse{}
		d.Info("targets:", targets, "len:", len(targets))
		for stackid := range targets {
			id := stackid
			iplist := strings.Split(targets[id].(string), ",")
			d.Info("target_stackid:", id)
			d.Info("ip:", iplist)
			scriptDir := os.TempDir() + "/sesha3/scripts/" + id
			if !private.Exists(scriptDir) {
				err := os.MkdirAll(scriptDir, os.ModePerm)
				notify.HookPost(errors.Wrap(err, "create scripts folder failed (fatal)"))
			}

			// create script file on sesha3 server
			scriptfile := scriptDir + "/" + getdata["script_name"].(string)
			err = ioutil.WriteFile(scriptfile, []byte(getdata["script"].(string)), 0755)
			err = os.Chmod(scriptfile, 0755)
			d.Info("scriptfile:", scriptfile)
			if err != nil {
				w.Write(sesha3.NewSimpleError(err).Marshal())
				notify.HookPost(err)
				return
			}

			d.Info("script created:", scriptfile)
			d.Info("id:", id)
			pemfile := os.TempDir() + "/user/" + id + ".pem"
			d.Info("pemfile:", pemfile)
			cmdData := make(map[string]interface{})
			cmdData["pem"] = pemfile
			cmdData["scriptfilepath"] = scriptfile
			cmdData["user"] = getdata["user"]
			cmdData["target"] = iplist
			cmdData["script_name"] = getdata["script_name"]
			d.Info("cmddata:", cmdData)
			// actual script execution
			out := execute.SshCmd(cmdData)
			resitem := sesha3.ExecScriptStackResponse{
				StackId: id,
				Outputs: out,
			}

			results = append(results, resitem)
		}

		payload, err := json.Marshal(results)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
		}

		w.Write(payload)
	*/
}

func noblank(str string) string {
	str = strings.Replace(str, "\r", "\n", -1)
	s := strings.Split(str, "\n")
	ret := []string{}
	for _, i := range s {
		if len(i) != 0 {
			ret = append(ret, i)
		}
	}
	return strings.Join(ret, "\n")
}

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

func version(w http.ResponseWriter, req *http.Request) {
	metrics.MetricsCurrentConnection.Add(1)
	defer metrics.MetricsCurrentConnection.Add(-1)
	w.Write([]byte(`{"version":"v0.0.15-dev"}`))
}

func redirect(w http.ResponseWriter, req *http.Request) {
	target := "https://" + req.Host + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}

	d.Info("redirect to:", target)
	http.Redirect(w, req, target, http.StatusMovedPermanently)
}
