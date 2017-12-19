package metrics

import (
	"expvar"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	as "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/golang/glog"
	"github.com/mobingilabs/sesha3/pkg/util"
)

var (
	MetricsMap               = expvar.NewMap("sesha3")
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
	valid      bool
	instanceId string
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
	n.region = util.GetRegion()
	n.valid = true
	n.instanceId = util.GetEc2Id()
	n.postMetrics()
	initTime, _ := time.ParseDuration("0ms")
	MetricsTTYResponseTime.Set(initTime.String())
	MetricsTokenResponseTime.Set(initTime.String())
}

func (n *HttpMetrics) postMetrics() {
	sess := as.Must(as.NewSessionWithOptions(as.Options{
		SharedConfigState: as.SharedConfigDisable,
	}))

	cli := cloudwatch.New(sess, &aws.Config{
		Region: aws.String(n.region),
	})

	go func() {
		// log only every 10th log
		infoCnt := 0
		for {
			time.Sleep(10 * time.Second)
			datums := n.GetCloudwatchPostData()
			req := &cloudwatch.PutMetricDataInput{
				Namespace:  aws.String("sesha3"),
				MetricData: datums,
			}

			_, err := cli.PutMetricData(req)
			if err != nil {
				glog.Errorf("putmetric failed: %v", err)
			}

			infoCnt += 1
			if infoCnt >= 9 {
				// d.Info("[10th] metrics sent to cloudwatch")
				infoCnt = 0
			}
		}
	}()
}

func (n *HttpMetrics) GetCloudwatchPostData() []*cloudwatch.MetricDatum {
	servername := "sesha3"
	data := []*cloudwatch.MetricDatum{}
	dimensionParam := &cloudwatch.Dimension{
		Name:  aws.String("PerInstance"),
		Value: aws.String(n.instanceId),
	}

	getDatumf := func(name string) (postdata *cloudwatch.MetricDatum) {
		timestamp := aws.Time(time.Now())
		sesha3Metrics := expvar.Get(servername).(*expvar.Map)
		switch v := sesha3Metrics.Get(name).(type) {
		case *expvar.Int:
			val := float64(v.Value())
			postdata = &cloudwatch.MetricDatum{
				MetricName: aws.String(name),
				Timestamp:  timestamp,
				Dimensions: []*cloudwatch.Dimension{dimensionParam},
				Value:      aws.Float64(val),
				Unit:       aws.String(cloudwatch.StandardUnitCount),
			}
		case *expvar.String:
			resString := v.String()
			val_tmp, _ := time.ParseDuration(resString)
			val := val_tmp.Seconds()
			postdata = &cloudwatch.MetricDatum{
				MetricName: aws.String(name),
				Timestamp:  timestamp,
				Dimensions: []*cloudwatch.Dimension{dimensionParam},
				Value:      aws.Float64(val),
				Unit:       aws.String(cloudwatch.StandardUnitMilliseconds),
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
