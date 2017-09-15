package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/syslog"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/mobingilabs/sesha3/token"
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
)

func ttyurl(w http.ResponseWriter, r *http.Request) {
	var sess session

	/*
		tokenerr, tokenmessage := token.GetToken(w, r)
		if tokenerr != true {
			w.Write([]byte(tokenmessage))
			return
		}
	*/

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	d.Info("req:", string(body))

	var m map[string]interface{}
	err = json.Unmarshal(body, &m)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	pemurl := m["pem"].(string)
	d.Info("rawurl:", pemurl)
	resp, err := http.Get(fmt.Sprintf("%v", pemurl))
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
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
		d.ErrorExit(err, 1)
	}

	pemfile := os.TempDir() + "/user/" + sess.StackId + ".pem"
	err = ioutil.WriteFile(pemfile, body, 0600)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	randomurl, err := sess.Start()
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	// add this session to our list of running sessions
	ttys.Add(sess)

	if randomurl == "" {
		w.Write(sesha3.NewSimpleError("cannot initialize secure tty access").Marshal())
		return
	} else {
		sess.Online = true
	}

	var fullurl string
	sess.TtyURL = randomurl
	fullurl = sess.GetFullURL()
	if fullurl == "" {
		w.Write(sesha3.NewSimpleError("cannot initialize secure tty access").Marshal())
		return
	}

	payload := `{"tty_url":"` + fullurl + `"}`
	w.Write([]byte(payload))
}
func version(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(`{"version":"v0.0.6-beta"}`))
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
	certfolder := cmdline.Dir() + "/certs"
	port := GetCliStringFlag(cmd, "port")
	router := mux.NewRouter()
	router.HandleFunc("/token", token.Settoken).Methods(http.MethodGet)
	router.HandleFunc("/ttyurl", ttyurl).Methods(http.MethodGet)
	router.HandleFunc("/version", version).Methods(http.MethodGet)
	err := http.ListenAndServeTLS(":"+port, certfolder+"/fullchain.pem", certfolder+"/privkey.pem", router)
	d.ErrorExit(err, 1)
}
