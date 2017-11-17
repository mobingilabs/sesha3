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

func (c *ApiController) Root() {
	c.Ctx.ResponseWriter.Write([]byte("Copyright (c) Mobingi. All rights reserved."))
}
