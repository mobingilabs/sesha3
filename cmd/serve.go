package cmd

import (
	"log/syslog"
	"net/http"
	"os"
	"path/filepath"
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
	"github.com/mobingilabs/sesha3/pkg/constants"
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
	if !private.Exists(constants.DATA_DIR) {
		err := os.MkdirAll(constants.DATA_DIR, 0700)
		if err != nil {
			glog.Errorf("mkdirall failed: %v", err)
			return err
		}
	}

	downloader := s3manager.NewDownloaderWithClient(svc)
	for _, i := range fnames {
		fl := filepath.Join(constants.DATA_DIR, i)
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

		glog.Infof("download s3 file: %v (bytes = %v)", fl, n)
	}

	return nil
}

func ServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "run as server",
		Long:  `Run as server.`,
		Run: func(cmd *cobra.Command, args []string) {
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
			glog.Infof("region: %v", util.GetRegion())

			// try setting up LetsEncrypt certificates locally
			err = cert.SetupLetsEncryptCert(true)
			if err != nil {
				notify.HookPost(err)
				glog.Exitf("setup letsencrypt failed: %v", err)
			} else {
				certfolder := filepath.Join("/etc/letsencrypt/live", util.Domain())
				glog.Infof("certificate folder: %v", certfolder)
			}

			startm := "--- server start ---\n"
			startm += "dns: " + util.GetPublicDns() + "\n"
			startm += "region: " + util.GetRegion() + "\n"
			startm += "ec2: " + util.GetEc2Id()
			notify.HookPost(startm)

			e := echo.New()

			// prep, should be the first middleware
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					cid := uuid.NewV4().String()
					c.Set("contextid", cid)
					c.Set("starttime", time.Now())

					// Helper func to print the elapsed time since this middleware. Good to call at end of
					// request handlers, right before/after replying to caller.
					c.Set("fnelapsed", func(ctx echo.Context) {
						start := ctx.Get("starttime").(time.Time)
						glog.Infof("<-- %v, %v %v, delta: %v",
							ctx.Get("contextid"),
							c.Request().URL.String(),
							c.Request().Method,
							time.Now().Sub(start))
					})

					glog.Infof("--> %v, %v %v",
						cid,
						c.Request().URL.String(),
						c.Request().Method)

					return next(c)
				}
			})

			e.Use(middleware.CORS())

			// print request information
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					glog.Infof("remoteaddr: %v", c.Request().RemoteAddr)
					glog.Infof("url rawquery: %v", c.Request().URL.RawQuery)
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
				return c.String(http.StatusOK, "Copyright (c) Mobingi, 2015-2017. All rights reserved.")
			})

			e.POST("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

			ep := api.New()
			e.POST("/token", ep.HandleHttpToken)
			e.POST("/ttyurl", ep.HandleHttpTtyUrl)
			e.POST("/exec", ep.HandleHttpExec)

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
