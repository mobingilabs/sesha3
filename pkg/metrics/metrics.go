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
	C_C        string `dynamo:"current_connection"`
	T_Req      string `dynamo:"token_req"`
	T_ReqCount string `dynamo:"token_req_count"`
	T_Res      string `dynamo:"token_responce"`
	Tty_Req    string `dynamo:"tty_req"`
	Tty_Res    string `dynamo:"tty_responce"`
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
	n.postMetrics()
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
			time.Sleep(30 * time.Second)
			sesha3Metrics := expvar.Get(servername).(*expvar.Map)
			evt := Event{
				ServerName: servername,
				C_Count:    sesha3Metrics.Get("connection_count").String(),
				C_C:        sesha3Metrics.Get("current_connection").String(),
				T_Req:      sesha3Metrics.Get("token_req").String(),
				T_ReqCount: sesha3Metrics.Get("token_req_count").String(),
				T_Res:      sesha3Metrics.Get("token_responce").String(),
				Tty_Req:    sesha3Metrics.Get("tty_req").String(),
				Tty_Res:    sesha3Metrics.Get("tty_responce").String(),
			}
			err := table.Put(evt).Run()
			if err != nil {
				d.Error(err)
			}
		}
	}()
}
