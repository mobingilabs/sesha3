package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/syslog"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/jwt"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	domain     string // set by cli flag
	port       string // set by cli flag
	region     string // set by cli flag
	ec2id      string // set by cli flag
	credprof   string // set by cli flag
	syslogging bool   // set by cli flag
	logger     *syslog.Writer
	notifier   sesha3.Notificate
)

func errcheck(v interface{}) {
	var err error
	switch v.(type) {
	case string:
		str := v.(string)
		if str != "" {
			err = notifier.WebhookNotification(str)
		}
	case error:
		terr := v.(error)
		if terr != nil {
			err = notifier.WebhookNotification(terr.Error())
		}
	default:
		str := fmt.Sprintf("%v", v)
		if str != "" {
			err = notifier.WebhookNotification(str)
		}
	}

	if err != nil {
		d.Error(errors.Wrap(err, "webhook notify failed"))
	}
}

func hookpost(v interface{}) {
	switch v.(type) {
	case string:
		err := v.(string)
		go errcheck(err)
	case error:
		err := v.(error)
		go errcheck(err)
	default:
		err := fmt.Sprintf("%v", v)
		go errcheck(err)
	}
}

func generateToken(w http.ResponseWriter, r *http.Request) {
	ctx, err := jwt.NewCtx()
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		hookpost(err)
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
}

func ttyurl(w http.ResponseWriter, r *http.Request) {
	sesha3.MetricsConnect.Add(1)
	defer sesha3.MetricsConnect.Add(-1)
	var sess session
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
	ok, err := checkToken(credprof, region, fmt.Sprintf("%s", u), md5p)
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
		hookpost(err)
		return
	}

	d.Info("token:", btoken)
	d.Info("body:", string(body))
	err = json.Unmarshal(body, &m)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		hookpost(err)
		return
	}

	pemurl := m["pem"].(string)
	d.Info("rawurl:", pemurl)
	resp, err := http.Get(fmt.Sprintf("%v", pemurl))
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		hookpost(err)
		return
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		hookpost(err)
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
			hookpost(err)
			return
		}
	}

	pemfile := os.TempDir() + "/user/" + sess.StackId + ".pem"
	err = ioutil.WriteFile(pemfile, body, 0600)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		hookpost(err)
		return
	}

	randomurl, err := sess.Start()
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		hookpost(err)
		return
	}

	// add this session to our list of running sessions
	ttys.Add(sess)
	if randomurl == "" {
		err := fmt.Errorf("%s", "cannot initialize secure tty access")
		w.Write(sesha3.NewSimpleError(err).Marshal())
		hookpost(err)
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
		hookpost(err)
		return
	}

	payload := `{"tty_url":"` + fullurl + `"}`
	w.Write([]byte(payload))
}

func describeSessions(w http.ResponseWriter, req *http.Request) {
	ds := ttys.Describe()
	b, err := json.Marshal(ds)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		hookpost(err)
		return
	}

	w.Write(b)
}

func version(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(`{"version":"v0.0.13-beta"}`))
}

func redirect(w http.ResponseWriter, req *http.Request) {
	target := "https://" + req.Host + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}

	d.Info("redirect to:", target)
	http.Redirect(w, req, target, http.StatusMovedPermanently)
}

func serve(cmd *cobra.Command) {
	// redirect every http request to https
	// go http.ListenAndServe(":80", http.HandlerFunc(redirect))
	// everything else will be https

	//check notification flags
	notificateArray, err := cmd.Flags().GetStringArray("notification")
	d.Info("serve:get notification flags", err)
	for _, i := range notificateArray {
		if i == "slack" {
			notifier.Slack = true
		}
	}

	hookpost("sesha3 server is started")

	certfolder := cmdline.Dir() + "/certs"
	port := GetCliStringFlag(cmd, "port")

	router := mux.NewRouter()
	router.HandleFunc("/token", generateToken).Methods(http.MethodGet)
	router.HandleFunc("/ttyurl", ttyurl).Methods(http.MethodGet)
	// router.HandleFunc("/sessions", describeSessions).Methods(http.MethodGet)
	router.HandleFunc("/version", version).Methods(http.MethodGet)
	router.Handle("/debug", sesha3.MetricsHandler)
	err = http.ListenAndServeTLS(":"+port, certfolder+"/fullchain.pem", certfolder+"/privkey.pem", router)
	d.ErrorExit(err, 1)
}
