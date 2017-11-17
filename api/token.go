package api

import (
	"encoding/json"
	"time"

	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/jwt"
	"github.com/mobingilabs/sesha3/pkg/metrics"
)

func handleHttpToken(c *ApiController) {
	start := time.Now()
	metrics.MetricsTokenRequestCount.Add(1)
	metrics.MetricsCurrentConnection.Add(1)
	defer metrics.MetricsCurrentConnection.Add(-1)
	metrics.MetricsTokenRequest.Add(1)
	defer metrics.MetricsTokenRequest.Add(-1)

	ctx, err := jwt.NewCtx()
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	/*
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}
	*/

	type creds_t struct {
		Username string `json:"username"`
		Password string `json:"passwd"`
	}

	var up creds_t
	err = json.Unmarshal(c.Ctx.Input.RequestBody, &up)
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	m := make(map[string]interface{})
	m["username"] = up.Username
	m["password"] = up.Password
	_, stoken, err := ctx.GenerateToken(m)
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	type _token_payload struct {
		Key string `json:"key"`
	}

	end := time.Now()
	metrics.MetricsTokenResponseTime.Set(end.Sub(start).String())
	c.Data["json"] = _token_payload{Key: stoken}
	c.ServeJSON()
}
