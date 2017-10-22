package exec

import (
	"encoding/json"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"io/ioutil"
	"net/http"
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
	type payload_t struct {
		Out string `json:"out"`
	}
	payload := payload_t{
		Out: "hello",
	}
	b, err := json.Marshal(payload)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}

	w.Write(b)
}
