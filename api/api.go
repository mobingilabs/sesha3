package api

import (
	"fmt"

	"github.com/astaxie/beego"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	uuid "github.com/satori/go.uuid"
)

type ApiController struct {
	beego.Controller
	skipPrepare bool
	noAuth      bool
	sessionId   string
}

func (c *ApiController) Prepare() {
	if c.skipPrepare {
		return
	}

	// do auth by default
	if !c.noAuth {
		debug.Info("base auth:")
	}

	if c.sessionId == "" {
		c.sessionId = fmt.Sprintf("%s", uuid.NewV4())
	}

	debug.Info("session:prepare:", c.sessionId)
}

func (c *ApiController) DispatchScratch() {
	type x struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	debug.Info("session:scratch:", c.sessionId)
	c.Data["json"] = x{Name: "foo", Value: "bar"}
	c.ServeJSON()
}

func (c *ApiController) DispatchRoot() {
	c.Ctx.ResponseWriter.Write([]byte("Copyright (c) Mobingi. All rights reserved."))
}

func (c *ApiController) DispatchToken() {
	handleHttpToken(c)
}

func (c *ApiController) DispatchTtyUrl() {
	handleHttpTtyUrl(c)
}

func (c *ApiController) DispatchExec() {
	handleHttpExec(c)
}
