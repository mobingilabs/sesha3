package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"log/syslog"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/cmdline"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/token"
	"github.com/spf13/cobra"
)

var (
	ctx        context
	domain     string // set by cli flag
	port       string // set by cli flag
	region     string // set by cli flag
	ec2id      string // set by cli flag
	credprof   string // set by cli flag
	syslogging bool   // set by cli flag
	logger     *syslog.Writer

/*
const (
	httpPort  = "80"
	awsRegion = "ap-northeast-1"

	devdomain  = "sesha3.labs.mobingi.com"
	devinst    = "i-0d6ff50d6caef8ffa"
	devprofile = "sesha3"

	//devdomain  = "testyuto.labs.mobingi.com"
	//devinst    = "i-09094885155fee296"
	//devprofile = "mobingi-yuto"
*/
)

/*
func getjson(w http.ResponseWriter, r *http.Request) interface{} {
	if r.Header.Get("Content-Type") != "application/json" {
		return 0
	}

	//To allocate slice for request body
	length, err := strconv.Atoi(r.Header.Get("Content-Length"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return 0
	}

	//Read body data to parse json
	body := make([]byte, length)
	length, err = r.Body.Read(body)
	if err != nil && err != io.EOF {
		w.WriteHeader(http.StatusInternalServerError)
		return 0
	}

	//parse json
	var jsonBody message
	err = json.Unmarshal(body[:length], &jsonBody)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return 0
	}
	w.WriteHeader(http.StatusOK)
	time.Sleep(2 * time.Second)
	return jsonBody
}

func sshkey(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	get, ok := getjson(w, r).(message)
	if !ok {
		return nil, errors.New("get input failed")
	}

	url := get.Pem
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "http get pem failed")
	}

	defer resp.Body.Close()
		byteArray, _ := ioutil.ReadAll(resp.Body)
		rsakey := string(byteArray)                      // stringで鍵を取得
		if strings.Index(rsakey, `AccessDenied`) != -1 { //URL judge
			get.Err = -1
			return get
		}

		_, err = os.Stat("./tmp/")

		if err == nil {
			log.Println("./tmp detected.")
		} else {
			os.Mkdir("./tmp", 0700)
		}
		ioutil.WriteFile("./tmp/"+get.Stackid+".pem", []byte(rsakey), 0600)
		return get

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "readall failed")
	}

	return b, nil
}
*/

func ttyurl(w http.ResponseWriter, r *http.Request) {
	var ctx context
	// var get message

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
	sbody := strings.Replace(string(body), `\`, "", -1)
	sbody = strings.Replace(string(body), `u0026`, "&", -1)
	err = json.Unmarshal([]byte(sbody), &m)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	pemurl := m["pem"]
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

	ctx.User = fmt.Sprintf("%v", m["user"])
	ctx.Ip = fmt.Sprintf("%v", m["ip"])
	ctx.StackId = fmt.Sprintf("%v", m["stackid"])
	ctx.Timeout = fmt.Sprintf("%v", m["timeout"])
	d.Info("ctx:", ctx)
	d.Info("pem:", string(body))
	pemdir := os.TempDir() + "/user/"
	_, err = os.Stat(pemdir)
	if err == nil {
		log.Println(pemdir + " detected.")
	} else {
		log.Println(pemdir + " not detected. mkdir" + pemdir)
		err = os.MkdirAll(pemdir, 0700)
		log.Println("mkdir err : ", err)
	}
	pemfile := os.TempDir() + "/user/" + ctx.StackId + ".pem"
	err = ioutil.WriteFile(pemfile, body, 0600)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	randomurl, err := ctx.Start()
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	if randomurl == "" {
		w.Write(sesha3.NewSimpleError("cannot initialize secure tty access").Marshal())
		return
	} else {
		ctx.Online = true
	}

	var fullurl string
	ctx.TtyURL = randomurl
	fullurl = ctx.GetFullURL()
	if fullurl == "" {
		w.Write(sesha3.NewSimpleError("cannot initialize secure tty access").Marshal())
		return
	}

	payload := `{"tty_url":"` + fullurl + `"}`
	w.Write([]byte(payload))
}
func version(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte(`{"version":"v0.0.5-beta"}`))
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
