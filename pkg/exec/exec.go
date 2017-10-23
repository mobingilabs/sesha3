package exec

import (
	"encoding/json"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

func sshcmd(data map[string]interface{}) {
	Ips := strings.Split(data["target"].(string), ",")
	for _, ip := range Ips {
		ssh := "/usr/bin/ssh -tt -oStrictHostKeyChecking=no -i " + os.TempDir() + "/user/" + data["stackid"].(string) + ".pem " + data["user"].(string) + "@" + ip
		d.Info(ssh)
	}
}

func Run(w http.ResponseWriter, r *http.Request) {
	var getdata map[string]interface{}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}
	d.Info("body:", string(body))
	err = json.Unmarshal(body, &getdata)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}
	scriptDir := os.TempDir() + "/sesha3/scripts/" + getdata["stackid"].(string)
	if !private.Exists(scriptDir) {
		err := os.MkdirAll(scriptDir, os.ModePerm)
		notify.HookPost(errors.Wrap(err, "create scripts folder failed (fatal)"))
	}

	//create script file on sesha3 server
	wfile, err := os.Create(scriptDir + "/" + getdata["script_name"].(string))
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}
	wfile.Write([]byte(getdata["script"].(string)))
	wfile.Close()
	//

	//ssh cmd
	sshcmd(getdata)
	// ...
	//

	//post response
	stdout := "stdout"
	stderr := "stderr"
	type payload_t struct {
		Out string `json:"stdout"`
		Err string `json:"stderr"`
	}
	payload := payload_t{
		Out: stdout,
		Err: stderr,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
	}
	w.Write(b)
	//
}
