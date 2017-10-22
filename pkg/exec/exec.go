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
)

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
	scriptDir := os.TempDir() + "/scripts/" + getdata["stackid"].(string)
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

	//scp
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
