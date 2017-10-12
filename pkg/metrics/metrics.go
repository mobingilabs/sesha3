package metrics

import (
	"expvar"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	as "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	d "github.com/mobingilabs/mobingi-sdk-go/pkg/debug"
	"github.com/mobingilabs/sesha3/pkg/params"
	"reflect"
	"strconv"
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
	region     string
	credprof   string
	valid      bool
	instanceID string
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
	n.instanceID = params.Ec2Id
	n.postMetrics()
}

func (n *HttpMetrics) postMetrics() {
	cred := credentials.NewSharedCredentials("/root/.aws/credentials", n.credprof)
	cli := cloudwatch.New(as.New(), &aws.Config{
		Region:      aws.String(n.region),
		Credentials: cred,
	})

	go func() {
		for {
			time.Sleep(10 * time.Second)
			datums := n.GetCloudwatchPostData()
			req := &cloudwatch.PutMetricDataInput{
				Namespace:  aws.String("seaha3"),
				MetricData: datums,
			}
			_, err := cli.PutMetricData(req)
			if err != nil {
				d.Error(err)
			}
		}
	}()
}

func (n *HttpMetrics) GetCloudwatchPostData() []*cloudwatch.MetricDatum {
	servername := "sesha3"
	data := []*cloudwatch.MetricDatum{}
	timestamp := aws.Time(time.Now())
	dimensionParam := &cloudwatch.Dimension{
		Name:  aws.String("Sesha3"),
		Value: aws.String(n.instanceID),
	}
	getDatumf := func(name string) *cloudwatch.MetricDatum {
		sesha3Metrics := expvar.Get(servername).(*expvar.Map)
		test := *sesha3Metrics.Get(name).(*expvar.Int)
		d.Info(reflect.TypeOf(test.Value()))
		val, _ := strconv.ParseFloat(sesha3Metrics.Get(name).String(), 64)
		return &cloudwatch.MetricDatum{
			MetricName: aws.String(name),
			Timestamp:  timestamp,
			Dimensions: []*cloudwatch.Dimension{dimensionParam},
			Value:      aws.Float64(val),
			Unit:       aws.String(cloudwatch.StandardUnitCount),
		}
	}

	data = append(data, getDatumf("connection_count"))
	data = append(data, getDatumf("current_connection"))
	data = append(data, getDatumf("token_req"))
	data = append(data, getDatumf("token_req_count"))
	data = append(data, getDatumf("tty_req"))
	return data
}
