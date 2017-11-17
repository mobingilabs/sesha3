package api

func handleHttpExec(c *ApiController) {
	/*
		var in sesha3.ExecScriptPayload

		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		d.Info("body:", string(body))
		err = json.Unmarshal(body, &in)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		// token check
		auth := strings.Split(r.Header.Get("Authorization"), " ")
		if len(auth) != 2 {
			w.WriteHeader(401)
			return
		}

		ctx, err := jwt.NewCtx()
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		btoken := auth[1]
		pt, err := ctx.ParseToken(btoken)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		nc := pt.Claims.(*jwt.WrapperClaims)
		u, _ := nc.Data["username"]
		p, _ := nc.Data["password"]
		d.Info("user:", u)
		md5p := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s", p))))
		ok, err := token.CheckToken(params.CredProfile, params.Region, fmt.Sprintf("%s", u), md5p)
		if !ok {
			w.WriteHeader(401)
			return
		}

		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			return
		}

		d.Info("token:", btoken)

		// pemfile download for ssh
		d.Info("pemurl:", in.Target.PemUrl)
		resp, err := http.Get(fmt.Sprintf("%v", in.Target.PemUrl))
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		defer resp.Body.Close()
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		workdir := os.TempDir() + "/" + in.Target.StackId + "_" + in.Target.Flag + "/"
		d.Info("workdir:", workdir)
		if !private.Exists(workdir) {
			err = os.MkdirAll(workdir, 0700)
			if err != nil {
				w.Write(sesha3.NewSimpleError(err).Marshal())
				notify.HookPost(err)
				return
			}

			d.Info("workdir created:", workdir)
		}

		pemfile := workdir + in.Target.Ip + ".pem"
		d.Info("pemfile:", pemfile)
		if !private.Exists(pemfile) {
			err = ioutil.WriteFile(pemfile, body, 0600)
			if err != nil {
				w.Write(sesha3.NewSimpleError(err).Marshal())
				notify.HookPost(err)
				return
			}

			d.Info("pemfile created:", pemfile)
		}

		// write script to temporary file
		script := workdir + "_runscript"
		err = ioutil.WriteFile(script, in.Script, 0755)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		err = os.Chmod(script, 0755)
		d.Info("script:", script)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
			return
		}

		d.Info("script created:", script)

		// actual script execution
		out := execute.SshCmd(execute.SshCmdInput{
			Ip:     in.Target.Ip,
			Pem:    pemfile,
			Script: script,
			VmUser: in.Target.VmUser,
		})

		sout := sesha3.ExecScriptStackResponse{
			StackId: in.Target.StackId,
			Outputs: []sesha3.ExecScriptInstanceResponse{out},
		}

		payload, err := json.Marshal(sout)
		if err != nil {
			w.Write(sesha3.NewSimpleError(err).Marshal())
			notify.HookPost(err)
		}

		d.Info("reply:", string(payload))
		w.Write(payload)
	*/
}
