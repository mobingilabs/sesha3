package api

import (
	"github.com/astaxie/beego"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
)

type BaseApiController struct {
	beego.Controller
}

func (c *BaseApiController) Prepare() {
	debug.Info("base prepare:")
}
