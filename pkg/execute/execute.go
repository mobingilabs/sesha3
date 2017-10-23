package execute

import (
	"bytes"
	"encoding/json"
	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

func execmd(cmd *exec.Cmd) (string, string, error) {
	var outb, errb bytes.Buffer
	var stdout, stderr string
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		d.Error(err)
	}
	stdout = outb.String()
	stderr = errb.String()
	return stdout, stderr, err
}

type result struct {
	stdout string
	stderr string
	ip     string
}

func sshcmd(data map[string]interface{}) []result {
	Ips := strings.Split(data["target"].(string), ",")
	pemfile := os.TempDir() + "/user/" + data["stackid"].(string) + ".pem "
	ret := []result{}

	var wg sync.WaitGroup
	for _, ip := range Ips {
		wg.Add(1)
		go func() {
			//scp sesha3 to user instance
			var out result
			out.ip = ip
			scp := exec.Command(
				"/usr/bin/scp",
				"-i", pemfile,
				os.TempDir()+"/sesha3/scripts/"+data["script_name"].(string),
				data["user"].(string)+"@"+ip+":/tmp/",
			)
			_, scpe, err := execmd(scp)
			if err != nil {
				out.stderr = scpe + "\n"
			}
			//

			execScript := exec.Command(
				"/usr/bin/ssh",
				"-tt",
				"-o",
				"StrictHostKeyChecking=no",
				"-i", pemfile,
				data["user"].(string)+"@"+ip,
				"/tmp/"+data["script_name"].(string),
			)
			scriptout, scripterr, err := execmd(execScript)
			out.stdout = out.stdout + scriptout + "\n"
			out.stderr = out.stderr + scripterr + "\n"
			ret = append(ret, out)
			wg.Done()
		}()
	}
	wg.Wait()
	return ret
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
	results := sshcmd(getdata)
	d.Info(results[0])
	// ...
	//

	//post response
	stdout := results[0].stdout
	stderr := results[0].stderr
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
