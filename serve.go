package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"syscall"
	"time"

	"github.com/gorilla/mux"
)

var (
	ctx context
)

const (
	devdomain    = "sesha3.labs.mobingi.com"
	domain       = "testyuto.labs.mobingi.com"
	httpPort     = "8080"
	profilename  = "mobingi-yuto"
	awsRegion    = "ap-northeast-1"
	devinst      = "i-0dc4d12c80f412f68"
	testinstance = "i-09094885155fee296"
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
	get := getjson(w, r).(message)
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
	var get message
	if r.Method == "POST" {
		w.WriteHeader(http.StatusBadRequest)
	}
	get = sshkey(w, r)
	if get.Err == -1 {
		w.Write([]byte(`{"error":"` + "AccessDenied. Key URL unenable." + `"}`))
		return
	}
	err := ctx.Start(get)
	if err != nil {
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}

	var fullurl string

	// wait for 5 seconds (at most) if we can get a tty url
	for i := 0; i < 5000; i++ {
		fullurl = ctx.GetFullURL()
		if fullurl != "" {
			break
		}
		time.Sleep(time.Millisecond * 1)
	}

	if fullurl == "" {
		w.Write([]byte(`{"error":"cannot initialize secure tty access"}`))
		return
	}

	payload := `{"tty_url":"` + fullurl + `"}`
	w.Write([]byte(payload))
}

func hook(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println("err:", err)
	}

	fmt.Println("[notification]")
	fmt.Println(string(body))

	var m map[string]interface{}
	err = json.Unmarshal(body, &m)

	if err == nil {
		u, ok := m["urls"]
		if ok {
			urls := u.([]interface{})
			u0 := fmt.Sprintf("%s", urls[0])
			log.Println("url:", u0)
			ctx.SetRandomURL(u0)
		}

		cu, ok := m["client_url"]
		if ok {
			url := fmt.Sprintf("%s", cu)
			log.Println("client url:", url)
			ctx.SetClientURL(url)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func signalHandler() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(
		sigchan,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	go func() {
		for {
			s := <-sigchan
			switch s {
			case syscall.SIGINT, syscall.SIGTERM:
				os.Exit(0)
			}
		}
	}()
}

func serve() {
	router := mux.NewRouter()
	router.HandleFunc("/json", tty)
	router.HandleFunc("/hook", hook).Methods(http.MethodPost)
	log.Fatal(http.ListenAndServe(":"+httpPort, router))
}
