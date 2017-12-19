package cmd

import (
	"fmt"
	"log"
	"log/syslog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/golang/glog"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/mobingilabs/mobingi-sdk-go/pkg/private"
	"github.com/mobingilabs/sesha3/api"
	"github.com/mobingilabs/sesha3/pkg/cert"
	"github.com/mobingilabs/sesha3/pkg/metrics"
	"github.com/mobingilabs/sesha3/pkg/notify"
	"github.com/mobingilabs/sesha3/pkg/params"
	"github.com/mobingilabs/sesha3/pkg/util"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"
)

var logger *syslog.Writer

func downloadTokenFiles() error {
	fnames := []string{"token.pem", "token.pem.pub"}
	bucket := "sesha3-prod"
	if params.IsDev {
		bucket = "sesha3"
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigDisable,
	}))

	svc := s3.New(sess, &aws.Config{
		Region: aws.String(util.GetRegion()),
	})

	// create dir if necessary
	tmpdir := os.TempDir() + "/jwt/rsa/"
	if !private.Exists(tmpdir) {
		err := os.MkdirAll(tmpdir, 0700)
		if err != nil {
			glog.Errorf("mkdirall failed: %v", err)
			return err
		}
	}

	downloader := s3manager.NewDownloaderWithClient(svc)
	for _, i := range fnames {
		fl := tmpdir + i
		f, err := os.Create(fl)
		if err != nil {
			glog.Errorf("create file failed: %v", err)
			return err
		}

		// write the contents of S3 Object to the file
		n, err := downloader.Download(f, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(i),
		})

		if err != nil {
			glog.Errorf("s3 download failed: %v", err)
			return err
		}

		glog.Infof("download s3 file: %v (bytes = %v", i, n)
	}

	return nil
}

func ServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "run as server",
		Long:  `Run as server.`,
		Run: func(cmd *cobra.Command, args []string) {
			if params.UseSyslog {
				logger, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_USER, "sesha3")
				if err != nil {
					notify.HookPost(errors.Wrap(err, "syslog setup failed"))
					glog.Exitf("syslog setup failed: %v", err)
				}

				log.SetFlags(0)
				log.SetPrefix("[" + util.GetEc2Id() + "] ")
				log.SetOutput(logger)
			}

			err := downloadTokenFiles()
			if err != nil {
				notify.HookPost(errors.Wrap(err, "download token files failed, fatal"))
				glog.Exitf("download token files failed: %v", err)
			}

			metrics.MetricsType.MetricsInit()
			eps, _ := cmd.Flags().GetStringArray("notify-endpoints")
			err = notify.Notifier.Init(eps)
			if err != nil {
				glog.Errorf("notifier init failed: %v", err)
			}

			glog.Infof("--- server start ---")
			glog.Infof("dns: %v:%v", util.GetPublicDns(), params.Port)
			glog.Infof("ec2: %v", util.GetEc2Id())
			glog.Infof("syslog: %v", params.UseSyslog)
			glog.Infof("region: %v", util.GetRegion())

			// try setting up LetsEncrypt certificates locally
			err = cert.SetupLetsEncryptCert(true)
			if err != nil {
				notify.HookPost(err)
				glog.Exitf("setup letsencrypt failed: %v", err)
			} else {
				certfolder := "/etc/letsencrypt/live/" + util.Domain()
				glog.Infof("certificate folder: %v", certfolder)
			}

			startm := "--- server start ---\n"
			startm += "dns: " + util.GetPublicDns() + "\n"
			startm += "region: " + util.GetRegion() + "\n"
			startm += "ec2: " + util.GetEc2Id() + "\n"
			startm += "syslog: " + fmt.Sprintf("%v", params.UseSyslog)
			notify.HookPost(startm)

			/*
				beego.BConfig.ServerName = constants.SERVER_NAME + ":1.0.0"
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
				beego.Handler("/debug/vars", metrics.MetricsHandler)

				// try enable cors
				beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
					AllowAllOrigins:  true,
					AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
					AllowHeaders:     []string{"Origin", "Authorization", "Access-Control-Allow-Origin", "Content-Type"},
					ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin"},
					AllowCredentials: true,
				}))

				beego.Run(":" + params.Port)
			*/

			e := echo.New()

			// time in, should be the first middleware
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					cid := uuid.NewV4().String()
					c.Set("contextid", cid)
					c.Set("starttime", time.Now())

					// Helper func to print the elapsed time since this middleware. Good to call at end of
					// request handlers, right before/after replying to caller.
					c.Set("fnelapsed", func(ctx echo.Context) {
						start := ctx.Get("starttime").(time.Time)
						glog.Infof("<-- %v, delta: %v", ctx.Get("contextid"), time.Now().Sub(start))
					})

					glog.Infof("--> %v", cid)
					return next(c)
				}
			})

			e.Use(middleware.CORS())

			// some information about request
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					glog.Infof("remoteaddr: %v", c.Request().RemoteAddr)
					return next(c)
				}
			})

			// add server name in response
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					c.Response().Header().Set(echo.HeaderServer, "mobingi:sesha3")
					return next(c)
				}
			})

			e.GET("/", func(c echo.Context) error {
				c.String(http.StatusOK, "Copyright (c) Mobingi, 2015-2017. All rights reserved.")
				return nil
			})

			ep := api.New()

			e.POST("/token", ep.HandleHttpToken)
			e.POST("/ttyurl", ep.HandleHttpTtyUrl)

			// serve
			glog.Infof("serving on :%v", params.Port)
			e.Server.Addr = ":" + params.Port
			gracehttp.Serve(e.Server)
		},
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&params.Port, "port", "8080", "server port")
	return cmd
}
