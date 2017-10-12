package metrics

import (
	"expvar"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	as "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
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
	initTime, _ := time.ParseDuration("0ms")
	MetricsTTYResponseTime.Set(initTime.String())
	MetricsTokenResponseTime.Set(initTime.String())
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
	dimensionParam := &cloudwatch.Dimension{
		Name:  aws.String("Sesha3"),
		Value: aws.String(n.instanceID),
	}
	getDatumf := func(name string) (postdata *cloudwatch.MetricDatum) {
		timestamp := aws.Time(time.Now())
		sesha3Metrics := expvar.Get(servername).(*expvar.Map)
		switch v := sesha3Metrics.Get(name).(type) {
		case *expvar.Int:
			d.Info(name)
			val := float64(v.Value())
			postdata = &cloudwatch.MetricDatum{
				MetricName: aws.String(name),
				Timestamp:  timestamp,
				Dimensions: []*cloudwatch.Dimension{dimensionParam},
				Value:      aws.Float64(val),
				Unit:       aws.String(cloudwatch.StandardUnitCount),
			}
		case *expvar.String:
			d.Info(name)
			resString := v.String()
			val_tmp, _ := time.ParseDuration(resString)
			val := val_tmp.Seconds()
			postdata = &cloudwatch.MetricDatum{
				MetricName: aws.String(name),
				Timestamp:  timestamp,
				Dimensions: []*cloudwatch.Dimension{dimensionParam},
				Value:      aws.Float64(val),
				Unit:       aws.String(cloudwatch.StandardUnitCountSecond),
			}
		}
		return
	}

	data = append(data, getDatumf("connection_count"))
	data = append(data, getDatumf("current_connection"))
	data = append(data, getDatumf("token_req"))
	data = append(data, getDatumf("token_req_count"))
	data = append(data, getDatumf("tty_req"))
	data = append(data, getDatumf("tty_responce"))
	data = append(data, getDatumf("token_responce"))
	return data
}
