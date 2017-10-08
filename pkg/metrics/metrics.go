package metrics

import (
	"expvar"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	as "github.com/aws/aws-sdk-go/aws/session"
	"github.com/guregu/dynamo"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/params"
	"time"
)

var (
	MetricsMap = expvar.NewMap("sesha3")

	MetricsConnectionCount   = new(expvar.Int)
	MetricsCurrentConnection = new(expvar.Int)
	MetricsTokenRequest      = new(expvar.Int)
	MetricsTokenRequestCount = new(expvar.Int)
	MetricsTokenResponseTime = new(expvar.String)
	MetricsTTYRequest        = new(expvar.Int)
	MetricsTTYResponseTime   = new(expvar.String)
	MetricsHandler           = expvar.Handler()
)

type Event struct {
	ServerName string `dynamo:"server_name"`
	C_Count    string `dynamo:"connection_count"`
	//	C_C        string `dynamo:"current_connection"`
}

type HttpMetrics struct {
	region   string
	credprof string
	valid    bool
}

var MetricsType HttpMetrics

func init() {
	MetricsMap.Set("connection_count", MetricsConnectionCount)
	MetricsMap.Set("current_connection", MetricsCurrentConnection)
	MetricsMap.Set("token_req", MetricsTokenRequest)
	MetricsMap.Set("token_req_count", MetricsTokenRequestCount)
	MetricsMap.Set("token_responce", MetricsTokenResponseTime)
	MetricsMap.Set("tty_req", MetricsTTYRequest)
	MetricsMap.Set("tty_responce", MetricsTTYResponseTime)
}

func (n *HttpMetrics) MetricsInit() {
	n.region = params.Region
	n.credprof = params.CredProfile
	n.valid = true
}

func (n *HttpMetrics) postMetrics() {
	servername := "sesha3"
	cred := credentials.NewSharedCredentials("/root/.aws/credentials", n.credprof)
	db := dynamo.New(as.New(), &aws.Config{
		Region:      aws.String(n.region),
		Credentials: cred,
	})

	table := db.Table("SESHA3_M")

	go func() {
		for {
			time.Sleep(10 * time.Second)
			sesha3Metrics := expvar.Get("sesha3").(*expvar.Map)
			evt := Event{
				ServerName: servername,
				C_Count:    sesha3Metrics.Get("connection_count").String(),
			}
			err := table.Put(evt).Run()
			if err != nil {
				d.Error(err)
			}
		}
	}()
}
