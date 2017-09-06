package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"time"

	"github.com/gorilla/mux"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	"github.com/mobingilabs/sesha3/token"
	"github.com/spf13/cobra"
)

var (
	ctx      context
	domain   string // set by cli flag
	port     string // set by cli flag
	region   string // set by cli flag
	ec2id    string // set by cli flag
	credprof string // set by cli flag
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

func getjson(w http.ResponseWriter, r *http.Request) interface{} {
	if r.Method != "POST" {
		return 0
	}

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

func sshkey(w http.ResponseWriter, r *http.Request) message {
	get, ok := getjson(w, r).(message)
	if !ok {
		get.Err = -1
		return get
	}
	url := get.Pem
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
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
		fmt.Println("./tmp detected.")
	} else {
		os.Mkdir("./tmp", 0700)
	}
	ioutil.WriteFile("./tmp/"+get.Stackid+".pem", []byte(rsakey), 0600)
	return get
}

func tty(w http.ResponseWriter, r *http.Request) {
	var ctx context
	var get message
	tokenerr, tokenmessage := token.GetToken(w, r)
	if tokenerr != true {
		w.Write([]byte(tokenmessage))
		return
	}

	if r.Method == "POST" {
		w.WriteHeader(http.StatusBadRequest)
	}
	get = sshkey(w, r)
	if get.Err == -1 {
		w.Write(sesha3.NewSimpleError("access denied, key url disabled").Marshal())
		return
	}
	randomurl, err := ctx.Start(get)
	if randomurl == "" {
		w.Write(sesha3.NewSimpleError("cannot initialize secure tty access").Marshal())
		return
	} else {
		ctx.Online = true
	}
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		return
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
	w.Write([]byte(`{"version":"v0.0.2-beta"}`))
}

func serve(cmd *cobra.Command) {
	port := GetCliStringFlag(cmd, "port")
	router := mux.NewRouter()
	router.HandleFunc("/token", token.Settoken)
	router.HandleFunc("/json", tty)
	router.HandleFunc("/version", version).Methods(http.MethodGet)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
