package api

import (
	"encoding/json"
	"time"

	"github.com/mobingilabs/mobingi-sdk-go/mobingi/sesha3"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/jwt"
	"github.com/mobingilabs/sesha3/pkg/metrics"
)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"passwd"`
}

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

	var creds credentials

	d.Info("body:", string(c.Ctx.Input.RequestBody))
	err = json.Unmarshal(c.Ctx.Input.RequestBody, &creds)
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	m := make(map[string]interface{})
	m["username"] = creds.Username
	m["password"] = creds.Password
	_, stoken, err := ctx.GenerateToken(m)
	if err != nil {
		c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
		return
	}

	end := time.Now()
	metrics.MetricsTokenResponseTime.Set(end.Sub(start).String())

	reply := make(map[string]string)
	reply["key"] = stoken
	c.Data["json"] = reply
	c.ServeJSON()
}
