package api

import (
	"github.com/astaxie/beego"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
)

type ApiController struct {
	beego.Controller
}

func (c *ApiController) Prepare() {
	debug.Info("base prepare:")
}

func (c *ApiController) DispatchScratch() {
	type x struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	c.Data["json"] = x{Name: "foo", Value: "bar"}
	c.ServeJSON()
}

func (c *ApiController) DispatchRoot() {
	c.Ctx.ResponseWriter.Write([]byte("Copyright (c) Mobingi. All rights reserved."))
}

func (c *ApiController) DispatchTtyUrl() {
	handleHttpTtyUrl(c)
}
