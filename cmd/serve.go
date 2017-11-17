package cmd

import (
	"fmt"
	"log"
	"log/syslog"
	"net/http"

	"github.com/astaxie/beego"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/api"
	"github.com/mobingilabs/sesha3/pkg/cert"
	"github.com/mobingilabs/sesha3/pkg/metrics"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/mobingilabs/sesha3/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type ServeCtx struct {
	localUrl string
}

var logger *syslog.Writer

func ServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "run as server",
		Long:  `Run as server.`,
		Run: func(cmd *cobra.Command, args []string) {
			if params.UseSyslog {
				logger, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, "sesha3")
				if err != nil {
					notify.HookPost(errors.Wrap(err, "syslog setup failed, fatal"))
					d.ErrorTraceExit(err, 1)
				}

				log.SetFlags(0)
				log.SetPrefix("[" + util.GetEc2Id() + "] ")
				log.SetOutput(logger)
			}

			metrics.MetricsType.MetricsInit()
			eps, _ := cmd.Flags().GetStringArray("notify-endpoints")
			err := notify.Notifier.Init(eps)
			if err != nil {
				d.Error(err)
			}

			d.Info("--- server start ---")
			d.Info("dns:", util.GetPublicDns()+":"+params.Port)
			d.Info("ec2:", params.Ec2Id)
			d.Info("syslog:", params.UseSyslog)
			d.Info("region:", params.Region)
			d.Info("credprof:", params.CredProfile)

			// try setting up LetsEncrypt certificates locally
			err = cert.SetupLetsEncryptCert(true)
			if err != nil {
				notify.HookPost(err)
				d.Error(err)
			} else {
				certfolder := "/etc/letsencrypt/live/" + util.Domain()
				d.Info("certificate folder:", certfolder)
			}

			startm := "--- server start ---\n"
			startm += "dns: " + util.GetPublicDns() + "\n"
			startm += "ec2: " + params.Ec2Id + "\n"
			startm += "syslog: " + fmt.Sprintf("%v", params.UseSyslog)
			notify.HookPost(startm)

			beego.BConfig.ServerName = "sesha3:1.0.0"
			beego.BConfig.RunMode = beego.PROD
			if params.IsDev {
				beego.BConfig.RunMode = beego.DEV
			}

			// needed for http input body in request to be available for non-get and head reqs
			beego.BConfig.CopyRequestBody = true

			beego.Router("/", &api.ApiController{}, "get:DispatchRoot")
			beego.Router("/scratch", &api.ApiController{}, "get:DispatchScratch")
			beego.Router("/token", &api.ApiController{}, "post:DispatchToken")
			beego.Router("/ttyurl", &api.ApiController{}, "post:DispatchTtyUrl")
			beego.Router("/exec", &api.ApiController{}, "post:DispatchExec")
			beego.Run(":" + params.Port)

			/*
				router := mux.NewRouter()
				router.HandleFunc("/", handleHttpRoot(c)).Methods(http.MethodGet)
				router.HandleFunc("/version", handleHttpVersion(c)).Methods(http.MethodGet)
				router.HandleFunc("/token", handleHttpToken(c)).Methods(http.MethodGet)
				router.HandleFunc("/ttyurl", handleHttpPtyUrl(c)).Methods(http.MethodGet)
				router.HandleFunc("/exec", handleHttpExecScript(c)).Methods(http.MethodGet)
				router.Handle("/debug/vars", metrics.MetricsHandler)

				// start our http server
				err = http.ListenAndServe(":"+params.Port, router)
				if err != nil {
					notify.HookPost(errors.Wrap(err, "server failed, fatal"))
					d.ErrorTraceExit(err, 1)
				}
			*/
		},
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&params.Port, "port", "8080", "server port")
	return cmd
}

/*
func describeSessions(w http.ResponseWriter, req *http.Request) {
	ds := session.Sessions.Describe()
	b, err := json.Marshal(ds)
	if err != nil {
		w.Write(sesha3.NewSimpleError(err).Marshal())
		notify.HookPost(err)
		return
	}

	w.Write(b)
}
*/

// TODO: use go generate to automate version
func handleHttpVersion(c *ServeCtx) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics.MetricsCurrentConnection.Add(1)
		defer metrics.MetricsCurrentConnection.Add(-1)
		w.Write([]byte(`{"version":"v0.0.15-dev"}`))
	})
}
