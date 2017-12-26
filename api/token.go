package api

/*
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
			notify.HookPost(err)
			c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
			c.err(errors.Wrap(err, "jwt ctx failed"))

			// if this fails, try force restart to redownload token files
			d.ErrorTraceExit(err, 1)
		}

		var creds credentials

		c.info("body:", string(c.Ctx.Input.RequestBody))
		err = json.Unmarshal(c.Ctx.Input.RequestBody, &creds)
		if err != nil {
			c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
			c.err(errors.Wrap(err, "unmarshal body failed"))
			return
		}

		m := make(map[string]interface{})
		m["username"] = creds.Username
		m["password"] = creds.Password
		tokenobj, stoken, err := ctx.GenerateToken(m)
		if err != nil {
			c.Ctx.ResponseWriter.Write(sesha3.NewSimpleError(err).Marshal())
			c.err(errors.Wrap(err, "generate token failed"))
			return
		}

		end := time.Now()
		metrics.MetricsTokenResponseTime.Set(end.Sub(start).String())

		c.info("token (obj):", tokenobj)
		c.info("token:", stoken)
		reply := make(map[string]string)
		reply["key"] = stoken
		c.Data["json"] = reply
		c.ServeJSON()
}
*/
