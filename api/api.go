package api

import (
	"encoding/json"
	"net/http"

	"github.com/astaxie/beego"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
)

type ApiController struct {
	beego.Controller
}

func (c *ApiController) Prepare() {
	debug.Info("base prepare:")
}

// decodeRequestBody decodes the http body to a Go object. It expects a pointer to
// an interface as parameter.
func (c *ApiController) decodeRequestBody(i interface{}) error {
	decoder := json.NewDecoder(c.Ctx.Input.Context.Request.Body)
	return decoder.Decode(i)
}

func (c *ApiController) json(i interface{}) {
	payload, err := json.Marshal(i)
	if err != nil {
		http.Error(c.Ctx.ResponseWriter, err.Error(), 500)
		debug.Error(err)
		return
	}

	c.Ctx.Output.Header("Content-Type", "application/json")
	c.Ctx.ResponseWriter.Write([]byte(payload))
}

func (c *ApiController) DispatchRoot() {
	c.Ctx.ResponseWriter.Write([]byte("Copyright (c) Mobingi. All rights reserved."))
}

func (c *ApiController) DispatchScratch() {
	c.Ctx.ResponseWriter.WriteHeader(500)
	type x struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	c.json(x{Name: "foo", Value: "bar"})
}

func (c *ApiController) DispatchTtyUrl() {
	handleHttpTtyUrl(c)
}
