package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
)

func TestToken(t *testing.T) {
	return
	if os.Getenv("TEST_TOKEN_PASS") != "" {
		start := time.Now()
		debug.Info("start:", start)
		type _cred struct {
			U string `json:"username"`
			P string `json:"passwd"`
		}

		cred := _cred{
			U: "subuser01",
			P: os.Getenv("TEST_TOKEN_PASS"),
		}

		b, _ := json.Marshal(cred)

		for i := 0; i < 1000; i++ {
			_start := time.Now()
			resp, err := http.Post("https://sesha3.demo.labs.mobingi.com/token", "application/json", bytes.NewBuffer(b))
			if err != nil {
				debug.Error(err)
			}

			debug.Info("["+time.Now().Sub(_start).String()+"]", resp)
		}

		debug.Info("end:", time.Now().Sub(start))
	}
}
